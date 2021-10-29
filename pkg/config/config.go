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
package config

import (
	"flag"
	"fmt"
	"io/ioutil"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	ConfigPath        *string
	LogLevel          *string
	LogPretty         *bool
	HTTPAddress       *string
	HTTPSAddress      *string
	MetricsAddress    *string
	SourceDir         *string
	DestinationDir    *string
	SyncAddress       *string
	SyncTimeout       *time.Duration
	SSLCrt            *string
	SSLKey            *string
	RedisEnabled      *bool
	RedisAddress      *string
	RedisPassword     *string
	RedisTLS          *bool
	RedisTLSInsecure  *bool
	ExecuteRedisQueue *bool
	SentryDSN         *string
}

const (
	syncTimeoutDefault = 30 * time.Second
)

var (
	gitVersion = "dev"
	appConfig  = Config{
		ConfigPath:        flag.String("config", getEnvDefault("CONFIG", "config.yaml"), "config"),
		LogPretty:         flag.Bool("log.pretty", false, "logging level"),
		LogLevel:          flag.String("log.level", "INFO", "logging level"),
		HTTPAddress:       flag.String("http.address", ":9336", "address"),
		HTTPSAddress:      flag.String("https.address", ":9335", "address"),
		MetricsAddress:    flag.String("metrics.address", ":9334", "address"),
		SourceDir:         flag.String("dir.src", "data", "folder"),
		DestinationDir:    flag.String("dir.dest", "data", "folder"),
		SyncAddress:       flag.String("sync.address", "localhost:9335", "destination server"),
		SyncTimeout:       flag.Duration("sync.timeout", syncTimeoutDefault, "destination server"),
		SentryDSN:         flag.String("sentry.dsn", os.Getenv("SENTRY_DSN"), "Sentry DSN"),
		SSLCrt:            flag.String("ssl.crt", "", "path to CA cert"),
		SSLKey:            flag.String("ssl.key", "", "path to CA key"),
		RedisEnabled:      flag.Bool("redis.enabled", false, "use redis"),
		RedisAddress:      flag.String("redis.address", "127.0.0.1:6379", "redis address"),
		RedisPassword:     flag.String("redis.password", "", "redis password"),
		RedisTLS:          flag.Bool("redis.tls", false, "use TLS in redis connection"),
		RedisTLSInsecure:  flag.Bool("redis.tls.insecure", false, "allow insecure tls connection"),
		ExecuteRedisQueue: flag.Bool("redis.executeQueue", true, "process redis queue, false in distributed mode"),
	}
)

func Load() error {
	config, err := ioutil.ReadFile(*appConfig.ConfigPath)
	if err != nil {
		return err
	}

	return yaml.Unmarshal(config, &appConfig)
}

func GetVersion() string {
	return gitVersion
}

func Get() *Config {
	return &appConfig
}

func String() string {
	out, err := yaml.Marshal(appConfig)
	if err != nil {
		return fmt.Sprintf("ERROR: %t", err)
	}

	return string(out)
}

func getEnvDefault(name string, defaultValue string) string {
	r := os.Getenv(name)
	defaultValueLen := len(defaultValue)

	if defaultValueLen == 0 {
		return r
	}

	if len(r) == 0 {
		return defaultValue
	}

	return r
}
