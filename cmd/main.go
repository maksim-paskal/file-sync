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
package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"time"

	logrushooksentry "github.com/maksim-paskal/logrus-hook-sentry"
	log "github.com/sirupsen/logrus"
)

//nolint:gochecknoglobals
var (
	gitVersion string = "dev"
)

func main() {
	ctx := context.Background()

	flag.Parse()

	if *appConfig.showVersion {
		//nolint:forbidigo
		fmt.Println(appConfig.Version)
		os.Exit(0)
	}

	level, err := log.ParseLevel(*appConfig.logLevel)
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.SetLevel(level)

	if !*appConfig.logPretty {
		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	}

	if log.GetLevel() == log.DebugLevel {
		log.SetReportCaller(true)
	}

	hook, err := logrushooksentry.NewHook(logrushooksentry.Options{
		SentryDSN: *appConfig.sentryDSN,
		Release:   appConfig.Version,
	})
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.AddHook(hook)

	defer hook.Stop()

	log.Infof("Starting %s...", appConfig.Version)

	api := newAPI()
	exporter := newExporter()
	queue := newQueue("file-sync")

	// for redis
	queue.onNewValue = func(message Message) {
		err := api.send(message)
		if err != nil {
			log.WithError(err).Error()
			exporter.queueErrorCounter.WithLabelValues(message.Type).Inc()

			return
		}
	}

	newWeb(exporter, queue, api).startServer()
	exporter.startServer()

	<-ctx.Done()
}
