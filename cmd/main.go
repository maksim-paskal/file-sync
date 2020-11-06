package main

import (
	"context"

	log "github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	config := newConfig()

	level, err := log.ParseLevel(*config.logLevel)
	if err != nil {
		log.Panic(err)
	}

	log.SetLevel(level)

	if log.GetLevel() == log.DebugLevel {
		log.SetReportCaller(true)
	}

	newWeb(config)

	<-ctx.Done()
}
