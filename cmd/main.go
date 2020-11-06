package main

import (
	"context"
	"flag"

	log "github.com/sirupsen/logrus"
)

func main() {
	ctx := context.Background()

	flag.Parse()

	level, err := log.ParseLevel(*appConfig.logLevel)
	if err != nil {
		log.Panic(err)
	}

	log.SetLevel(level)

	if !*appConfig.logPretty {
		log.SetFormatter(&log.JSONFormatter{})
	}

	if log.GetLevel() == log.DebugLevel {
		log.SetReportCaller(true)
	}

	newWeb()

	<-ctx.Done()
}
