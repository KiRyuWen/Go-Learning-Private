package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"shrimpbot/internal/ai"
	"shrimpbot/internal/crawler"
	"shrimpbot/internal/queue"
	"sync"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/redis/go-redis/v9"
	"google.golang.org/genai"
)

type ClientData struct {
	RedisClient  *redis.Client
	GeminiClient *genai.Client
}

func handleResultMessage(err error, s *discordgo.Session, job *queue.DiscordMessage) {
	if err != nil {
		log.Println(err.Error())
		s.ChannelMessageEdit(job.ChannelID, job.EditMsgID, err.Error())
	} else {
		s.ChannelMessageDelete(job.ChannelID, job.EditMsgID)
	}
}

func sendMessage(message string, s *discordgo.Session, job *queue.DiscordMessage) (err error) {
	msgRef := &discordgo.MessageSend{
		Reference: &discordgo.MessageReference{
			MessageID: job.CommandMsgID,
			ChannelID: job.ChannelID,
		},
	}
	toSends := ProcessMessage(message)
	for _, msg := range toSends {
		msgRef.Content = msg
		_, err := s.ChannelMessageSendComplex(job.ChannelID, msgRef)
		if err != nil {
			return err
		}
	}
	return nil
}

func ProcessMessage(msg string) []string {
	msgInRunes := []rune(msg)
	results := []string{}
	start := 0
	end := len(msgInRunes)
	log.Printf("Process maximum words: %d\n", end)
	for start < end {
		dst := end - start
		log.Printf("start: %d \tDst: %d\n", start, dst)

		if dst <= 2000 {
			results = append(results, string(msgInRunes[start:start+dst]))
			break
		}
		dst = 2000
		findNewline := false
		for j := dst + start; j > start; j-- {
			if msgInRunes[j] == '\n' {
				results = append(results, string(msgInRunes[start:j+1]))
				dst = j - start + 1
				findNewline = true
				break
			}
		}
		if !findNewline {
			results = append(results, string(msgInRunes[start:start+dst]))
		}
		start += dst
	}

	return results
}

func runWorker(ctx context.Context, s *discordgo.Session, clientData *ClientData, workerId int) {
	log.Printf("Worker %d starts", workerId)
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping\n", workerId)
			return
		default:
		}

		result, err := clientData.RedisClient.BLPop(ctx, time.Second, queue.DiscordTaskQueue).Result()
		if err != nil {
			switch {
			case err == context.Canceled:
				log.Println("Key does not exist")
			case err != nil:
				time.Sleep(time.Second)
				continue
			default:
				log.Println("Unkknown error")
				time.Sleep(time.Second)
			}
		}

		payload := result[1]

		var job queue.DiscordMessage
		json.Unmarshal([]byte(payload), &job)
		log.Printf("Worker %d starts working on Job %s\n", workerId, job.ID)
		switch job.Type {
		case queue.JobTypeBahamut:
			title, content, err := crawler.ScrapeBahamut(job.TargetURL)
			if err != nil {
				log.Printf("Error when scraping %s", err.Error())
				s.ChannelMessageEdit(job.ChannelID, job.EditMsgID, err.Error())
				continue
			}
			message := fmt.Sprintf("%s\n\n%s", title, content)
			err = sendMessage(message, s, &job)
			handleResultMessage(err, s, &job)
		case queue.JobTypeYouTube:
			s.ChannelMessageEdit(job.ChannelID, job.EditMsgID, "The video is processing, and it will take a while!!")
			output, err := ai.SummarizeVideo(ctx, clientData.GeminiClient, job.TargetURL)
			if err != nil {
				log.Printf("Error when understanding video %s", err.Error())
				s.ChannelMessageEdit(job.ChannelID, job.EditMsgID, err.Error())
				continue
			}
			err = sendMessage(output, s, &job)
			handleResultMessage(err, s, &job)
		default:
			log.Println("Invalid type", job)
		}

		log.Printf("Worker %d Done\n", workerId)
	}
}

func StartWorkerPool(ctx context.Context, s *discordgo.Session, clientData *ClientData, workerCount int, wg *sync.WaitGroup) {
	log.Println("Enable Routine StartWorker")
	for i := 1; i <= workerCount; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			runWorker(ctx, s, clientData, i)
		}(i)
	}
}
