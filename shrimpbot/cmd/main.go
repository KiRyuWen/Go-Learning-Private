package main

import (
	"fmt"
	"log"
	"net/url"
	"os"
	"os/signal"
	"shrimpbot/internal/crawler"
	"strings"
	"syscall"

	"github.com/bwmarrin/discordgo"
	"github.com/joho/godotenv"
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
	// Register the messageCreate func as a callback for MessageCreate events.
	dg.AddHandler(messageCreate)

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

	// Cleanly close down the Discord session.
	dg.Close()
}

func ProcessMessage(msg string) {
	msgInRunes := []rune(message)

	if len(msgInRunes) > 2000 {
		length := len(msgInRunes)
		splitMsgs := [][]rune{}
		//TODO: split messages within 2000 words, for each 2000 wordsparagraph, you need to split the message without abrupt word
		//In other words, you need to find the newline within 2000 words, and then start from there to find a closest 2000 words paragraph

	}
}

// This function will be called (due to AddHandler above) every time a new
// message is created on any channel that the authenticated bot has access to.
func messageCreate(s *discordgo.Session, m *discordgo.MessageCreate) {

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
	log.Println("Received URI")

	if strings.Contains(text, "https://forum.gamer.com.tw/") {
		log.Println("Receive Bahamut message")
		// parse specific url to bsn
		title, content, err := crawler.ScrapeBahamut(text)
		if err != nil {
			log.Printf("Error when scraping %s", err.Error())
			return
		}
		message := fmt.Sprintf("%s\n\n%s", title, content)
		msgInRunes := []rune(message)

		s.ChannelMessageSend(m.ChannelID, message)
	}

}
