package main

import (
	"context"
	"flag"

	log "github.com/sirupsen/logrus"
)

//nolint:gochecknoglobals
var (
	gitVersion string = "dev"
	buildTime  string
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

	log.Infof("Starting %s...", appConfig.Version)

	api := newAPI()
	exporter := newExporter()
	queue := newQueue("file-sync")

	// for redis
	queue.onNewValue = func(message Message) {
		err := api.send(message)
		if err != nil {
			log.Error(err)
			exporter.queueErrorCounter.WithLabelValues(message.Type).Inc()

			return
		}
	}

	newWeb(exporter, queue, api).startServer()
	exporter.startServer()

	<-ctx.Done()
}
