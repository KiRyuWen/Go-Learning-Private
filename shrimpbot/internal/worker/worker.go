package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"shrimpbot/internal/crawler"
	"shrimpbot/internal/queue"
	"time"

	"github.com/bwmarrin/discordgo"
	"github.com/redis/go-redis/v9"
)

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

func runWorker(ctx context.Context, s *discordgo.Session, rdsClient *redis.Client, workerId int) {
	log.Printf("Worker %d starts", workerId)
	for {
		select {
		case <-ctx.Done():
			log.Printf("Worker %d stopping\n", workerId)
			return
		default:
		}

		result, err := rdsClient.BLPop(ctx, time.Second, queue.DiscordTaskQueue).Result()
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

		switch job.Type {
		case queue.JobTypeBahamut:
			title, content, err := crawler.ScrapeBahamut(job.TargetURL)
			if err != nil {
				log.Printf("Error when scraping %s", err.Error())
				s.ChannelMessageSend(job.ChannelID, err.Error())
				continue
			}
			message := fmt.Sprintf("%s\n\n%s", title, content)
			toSends := ProcessMessage(message)
			for _, msg := range toSends {
				_, err := s.ChannelMessageSend(job.ChannelID, msg)
				if err != nil {
					log.Println(err)
					continue
				}
			}
		case queue.JobTypeYouTube:
			s.ChannelMessageSend(job.ChannelID, "Not implemented")
		default:
			log.Println("Invalid type", job)
		}

		log.Printf("Worker %d Done\n", workerId)
	}
}

func StartWorkerPool(ctx context.Context, s *discordgo.Session, rdsClient *redis.Client, workerCount int) {
	log.Println("Enable Routine StartWorker")
	for i := 1; i <= workerCount; i++ {
		go func(workerID int) {
			runWorker(ctx, s, rdsClient, i)
		}(i)
	}
}
