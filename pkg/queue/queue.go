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
	"sync"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/google/uuid"
	"github.com/maksim-paskal/file-sync/pkg/api"
	"github.com/maksim-paskal/file-sync/pkg/config"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const key = "file-sync"

var (
	rdb        *redis.Client
	OnNewValue func(api.Message)
	mutex      sync.Mutex
	ctx        = context.Background()
	mustStop   = false
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

func GracefullShutdown() {
	mustStop = true
}

func executeRedisQueue(ctx context.Context) {
	for {
		if mustStop {
			break
		}

		result, err := rdb.BLPop(ctx, 0*time.Second, key).Result()
		if err != nil {
			log.WithError(err).Error("error in queue.rdb.BLPop")

			continue
		}

		message := api.Message{}

		err = json.Unmarshal([]byte(result[1]), &message)
		if err != nil {
			log.
				WithError(err).
				WithField("message", message.String()).
				Error("error in json.Unmarshal")

			continue
		}

		onNewValue(message)
	}
}

func onNewValue(message api.Message) {
	mutex.Lock()
	defer mutex.Unlock()

	if OnNewValue != nil {
		OnNewValue(message)
	}
}

func Add(value api.Message) (string, error) {
	if len(value.ID) == 0 {
		value.ID = uuid.NewString()
	}

	messageJSON, err := json.Marshal(value)
	if err != nil {
		return value.ID, errors.Wrap(err, "error in json.Marshal")
	}

	push := rdb.RPush(ctx, key, messageJSON)

	_, err = push.Result()
	if err != nil {
		return value.ID, errors.Wrap(err, "error push.Result")
	}

	return value.ID, err
}

type ListResult struct {
	ID   string
	File string
	Type string
}

func Info() (string, error) {
	return rdb.Info(ctx).Result()
}
