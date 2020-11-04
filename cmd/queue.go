package main

import (
	"context"
	"log"
	"time"

	"github.com/go-redis/redis/v8"
)

type Queue struct {
	rdb        *redis.Client
	key        string
	onNewValue func(string)
	config     *Config
}

func newQueue(config *Config, key string) *Queue {
	queue := Queue{
		key:    key,
		config: config,
	}

	ctx := context.Background()
	queue.rdb = redis.NewClient(&redis.Options{
		Addr: *config.RedisAddress,
	})

	err := queue.rdb.FlushDB(ctx).Err()
	if err != nil {
		panic(err)
	}

	go func() {
		for {
			result, err := queue.rdb.BLPop(ctx, 0*time.Second, queue.key).Result()
			if err != nil {
				log.Fatal(err)
			}

			queue.onNewValue(result[1])
		}
	}()

	return &queue
}

func (q *Queue) add(value string) {
	ctx := context.Background()

	q.rdb.RPush(ctx, q.key, value)
}
