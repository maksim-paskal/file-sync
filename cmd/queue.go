package main

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	log "github.com/sirupsen/logrus"
)

type Queue struct {
	rdb        *redis.Client
	key        string
	onNewValue func(Message)
	mutex      sync.Mutex
}

func newQueue(key string) *Queue {
	queue := Queue{
		key: key,
	}

	// queue work with redis only
	if !*appConfig.redisEnabled {
		return &Queue{}
	}

	ctx := context.Background()
	queue.rdb = redis.NewClient(&redis.Options{
		Addr: *appConfig.redisAddress,
	})

	err := queue.rdb.FlushDB(ctx).Err()
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("Redis queue started on %s server", *appConfig.redisAddress)

	if *appConfig.executeRedisQueue {
		go func() {
			for {
				result, err := queue.rdb.BLPop(ctx, 0*time.Second, queue.key).Result()
				if err != nil {
					log.Error(err)
				}

				message := Message{}

				err = json.Unmarshal([]byte(result[1]), &message)
				if err != nil {
					log.Error(err)
				}

				// run command in same order
				queue.onNewValueSync(message)
			}
		}()
	}

	return &queue
}

func (q *Queue) onNewValueSync(message Message) {
	q.mutex.Lock()
	defer q.mutex.Unlock()

	if q.onNewValue != nil {
		q.onNewValue(message)
	}
}

func (q *Queue) add(value Message) (int64, error) {
	ctx := context.Background()

	messageJSON, err := json.Marshal(value)
	if err != nil {
		return -1, err
	}

	push := q.rdb.RPush(ctx, q.key, messageJSON)

	return push.Result()
}