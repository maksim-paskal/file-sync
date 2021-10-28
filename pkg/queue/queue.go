/*
Copyright paskal.maksim@gmail.com
Licensed under the Apache License, Version 2.0 (the "License")
you may not use this file except in compliance with the License.
You may obtain a copy of the License at
http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
package queue

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"strconv"
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/maksim-paskal/file-sync/pkg/api"
	"github.com/maksim-paskal/file-sync/pkg/config"
	"github.com/maksim-paskal/file-sync/pkg/metrics"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	key                = "file-sync"
	queueTimeout       = time.Second
	queueSize          = 10
	retryTimeRatio     = 2
	decimialRateNumber = 10
	maxListSize        = 100
	schedulePeriod     = 5 * time.Second
)

var (
	rdb        *redis.Client
	OnNewValue func(api.Message)
	mutex      sync.Mutex
	ctx        = context.Background()
)

func Init() error {
	// queue work with redis only
	if !*config.Get().RedisEnabled {
		return nil
	}

	redisOptions := redis.Options{
		Addr:     *config.Get().RedisAddress,
		Password: *config.Get().RedisPassword,
	}

	if *config.Get().RedisTLS {
		redisOptions.TLSConfig = &tls.Config{
			MinVersion: tls.VersionTLS12,
		}

		if *config.Get().RedisTLSInsecure {
			redisOptions.TLSConfig.InsecureSkipVerify = true
		}
	}

	rdb = redis.NewClient(&redisOptions)

	log.Infof("Redis queue started on %s server", *config.Get().RedisAddress)

	if *config.Get().ExecuteRedisQueue {
		go executeRedisQueue(ctx)
	}

	return nil
}

func Flush() error {
	return rdb.FlushDB(ctx).Err()
}

func executeRedisQueue(ctx context.Context) {
	for {
		time.Sleep(queueTimeout)

		timeNow := strconv.FormatInt(time.Now().Unix(), decimialRateNumber)

		results, err := rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
			Min:   "0",
			Max:   timeNow,
			Count: queueSize,
		}).Result()
		if err != nil {
			log.WithError(err).Error("error in queue.rdb.ZRangeByScore")

			continue
		}

		if len(results) == 0 {
			continue
		}

		_, err = rdb.ZRem(ctx, key, results).Result()
		if err != nil {
			log.WithError(err).Error("error in queue.rdb.ZRem")

			continue
		}

		for _, result := range results {
			message := api.Message{}

			err = json.Unmarshal([]byte(result), &message)
			if err != nil {
				log.
					WithError(err).
					WithField("message", message).
					Error("error in json.Unmarshal")

				continue
			}

			// run command in same order
			onNewValue(message)
		}
	}
}

func onNewValue(message api.Message) {
	mutex.Lock()
	defer mutex.Unlock()

	if OnNewValue != nil {
		OnNewValue(message)
	}
}

func delayTimeInSeconds(seconds int) int64 {
	delay := time.Duration(seconds) * time.Second

	return time.Now().Add(delay).Unix()
}

func Add(value api.Message) (string, error) {
	if len(value.ID) == 0 {
		value.ID = uuid.NewString()
	}

	messageJSON, err := json.Marshal(value)
	if err != nil {
		return value.ID, errors.Wrap(err, "error in json.Marshal")
	}

	retryTime := delayTimeInSeconds(value.RetryCount * retryTimeRatio)

	_, err = rdb.ZAdd(ctx, key, &redis.Z{
		Score:  float64(retryTime),
		Member: messageJSON,
	}).Result()

	return value.ID, err
}

type ListResult struct {
	ID         string
	File       string
	Type       string
	RetryCount int
	Error      string
}

func List() ([]ListResult, error) {
	results, err := rdb.ZRangeByScore(ctx, key, &redis.ZRangeBy{
		Min:   "0",
		Max:   "+inf",
		Count: maxListSize,
	}).Result()
	if err != nil {
		return nil, err
	}

	listMessages := make([]ListResult, 0)

	for _, result := range results {
		message := api.Message{}

		err = json.Unmarshal([]byte(result), &message)
		if err != nil {
			return nil, err
		}

		listMessages = append(listMessages, ListResult{
			ID:         message.ID,
			File:       message.FileName,
			Type:       message.Type,
			RetryCount: message.RetryCount,
			Error:      message.RetryLastError,
		})
	}

	return listMessages, nil
}

func Info() (string, error) {
	return rdb.Info(ctx).Result()
}

// Shedule collecting metrics.
func ScheduleMetrics() {
	for {
		queueLen, err := List()
		if err != nil {
			log.WithError(err).Error()
		}

		metrics.QueueSize.Set(float64(len(queueLen)))
		time.Sleep(schedulePeriod)
	}
}
