package main

import (
	"context"
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"shrimpbot/internal/ai"
	"shrimpbot/internal/queue"
	"shrimpbot/internal/worker"
	"sync"
	"syscall"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"
)

func main() {
	err := godotenv.Load()

	if err != nil {
		log.Fatal("Unable to read .env file")
	}
	token := os.Getenv("BOT_TOKEN")

	dg, err := discordgo.New("Bot " + token)
	if err != nil {
		fmt.Println("error creating Discord session,", err)
		return
	}

	redisClient, err := queue.InitRedis()
	if err != nil {
		log.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())

	geminiClient, err := ai.InitGemini(ctx)
	if err != nil {
		log.Fatal(err)
	}

	clientData := &worker.ClientData{
		RedisClient:  redisClient,
		GeminiClient: geminiClient,
	}

	var wg sync.WaitGroup

	worker.StartWorkerPool(ctx, dg, clientData, 5, &wg)

	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(func(s *discordgo.Session, m *discordgo.MessageCreate) {
		messageCreate(s, m, ctx, redisClient)
	})

	// In this example, we only care about receiving message events.
	dg.Identify.Intents = discordgo.IntentsGuildMessages

	// Open a websocket connection to Discord and begin listening.
	err = dg.Open()
	if err != nil {
		fmt.Println("error opening connection,", err)
		return
	}

	// Wait here until CTRL-C or other term signal is received.
	fmt.Println("Bot is now running.  Press CTRL-C to exit.")
	sc := make(chan os.Signal, 1)
	signal.Notify(sc, syscall.SIGINT, syscall.SIGTERM, os.Interrupt)
	<-sc

	cancel() // Notify the worker to stop
	isWorkerDone := make(chan struct{})
	go func() {
		wg.Wait()
		close(isWorkerDone)
	}()

	// Cleanly close down the Discord session.
	dg.Close()

	select {
	case <-isWorkerDone:
		fmt.Println("Worker finished gracefully")
	case <-time.After(1 * time.Minute):
		fmt.Println("Enforcing shutdown")
	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate, ctx context.Context, rdsClient *redis.Client) {

	// Ignore all messages created by the bot itself
	// This isn't required in this specific example but it's a good practice.
	if m.Author.ID == s.State.User.ID {
		return
	}

	text := m.Content
	_, err := url.ParseRequestURI(text)
	if err != nil {
		return
	}

	msgType, isValid := queue.GetMsgType(text)

	if !isValid {
		return
	}
	log.Println("Received URI")

	discordMsg := queue.NewDiscordMessage(m.ID, msgType, m.ChannelID, text)
	msgResp, err := s.ChannelMessageSend(m.ChannelID, "Receive Message, start working!")
	discordMsg.EditMsgID = msgResp.ID

	err = queue.Enqueue(ctx, rdsClient, discordMsg)
	if err != nil {
		s.ChannelMessageEdit(m.ChannelID, msgResp.ID, err.Error())
	}

}
