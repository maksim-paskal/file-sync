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

	queue := newQueue(config, "test")
	api := newAPI(config)

	queue.onNewValue = func(value string) {
		log.Infof("new=%s", value)

		message := Message{
			Type:        "PUT",
			FileName:    "aaa.txt",
			FileContent: "aaaaa",
		}

		err := api.makeTLSCall(message)
		if err != nil {
			log.Error(err)
		}
	}

	newWeb(config, queue)

	<-ctx.Done()
}
