package main

import (
	"flag"
	"fmt"
	"time"
)

type Config struct {
	Version           string
	logLevel          *string
	logPretty         *bool
	httpAddress       *string
	httpsAddress      *string
	metricsAddress    *string
	sourceDir         *string
	destinationDir    *string
	syncAddress       *string
	syncTimeout       *time.Duration
	sslClientKey      *string
	sslClientCrt      *string
	sslServerKey      *string
	sslServerCrt      *string
	sslCA             *string
	redisEnabled      *bool
	redisAddress      *string
	executeRedisQueue *bool
}

const (
	syncTimeoutDefault = 30 * time.Second
)

//nolint:gochecknoglobals
var appConfig Config = Config{
	Version:        fmt.Sprintf("%s-%s", gitVersion, buildTime),
	logPretty:      flag.Bool("log.pretty", false, "logging level"),
	logLevel:       flag.String("log.level", "INFO", "logging level"),
	httpAddress:    flag.String("http.address", ":9336", "address"),
	httpsAddress:   flag.String("https.address", ":9335", "address"),
	metricsAddress: flag.String("metrics.address", ":9334", "address"),
	sourceDir:      flag.String("dir.src", "data", "folder"),
	destinationDir: flag.String("dir.dest", "data", "folder"),
	syncAddress:    flag.String("sync.address", "localhost:9335", "destination server"),
	syncTimeout:    flag.Duration("sync.timeout", syncTimeoutDefault, "destination server"),
	// ssl config
	sslClientKey:      flag.String("ssl.clientKey", "ssl/client01.key", "ssl certificate"),
	sslClientCrt:      flag.String("ssl.clientCrt", "ssl/client01.crt", "ssl certificate"),
	sslServerKey:      flag.String("ssl.serverKey", "ssl/server.key", "ssl certificate"),
	sslServerCrt:      flag.String("ssl.serverCrt", "ssl/server.crt", "ssl certificate"),
	sslCA:             flag.String("ssl.serverCA", "ssl/ca.crt", "ssl certificate"),
	redisEnabled:      flag.Bool("redis.enabled", false, "use redis"),
	redisAddress:      flag.String("redis.address", "127.0.0.1:6379", "redis address"),
	executeRedisQueue: flag.Bool("redis.executeQueue", true, "process redis queue, false in distributed mode"),
}
