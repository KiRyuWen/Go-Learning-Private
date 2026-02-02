package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/redis/go-redis/v9"
)

func InitRedis() (client *redis.Client, err error) {
	addr := os.Getenv("REDIS_IP")
	port := os.Getenv("REDIS_PORT")

	client = redis.NewClient(&redis.Options{
		Addr: fmt.Sprintf("%s:%s", addr, port),
	})

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()

	_, err = client.Ping(pingCtx).Result()
	if err != nil {
		return nil, err
	}

	return client, err
}

func Enqueue(ctx context.Context, rdb *redis.Client, msg *DiscordMessage) (err error) {
	log.Println("Enqueue start")
	if rdb == nil {
		return fmt.Errorf("Redis is empty")
	}
	timeoutCtx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	payload, err := json.Marshal(*msg)
	if err != nil {
		return err
	}

	return rdb.RPush(timeoutCtx, DiscordTaskQueue, payload).Err()
}
