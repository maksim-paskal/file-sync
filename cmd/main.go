package main

import (
	"context"

	log "github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	config := newConfig()

	log.SetLevel(log.InfoLevel)

	if log.GetLevel() == log.DebugLevel {
		log.SetReportCaller(true)
	}

	newWeb(config)

	<-ctx.Done()
}
