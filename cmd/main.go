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
	"os"
	"time"

	"github.com/maksim-paskal/file-sync/pkg/api"
	"github.com/maksim-paskal/file-sync/pkg/config"
	"github.com/maksim-paskal/file-sync/pkg/metrics"
	"github.com/maksim-paskal/file-sync/pkg/queue"
	"github.com/maksim-paskal/file-sync/pkg/web"
	logrushooksentry "github.com/maksim-paskal/logrus-hook-sentry"
	log "github.com/sirupsen/logrus"
)

var showVersion = flag.Bool("version", false, "get version")

func main() { //nolint: cyclop
	ctx := context.Background()

	flag.Parse()

	if *showVersion {
		os.Stdout.WriteString(config.GetVersion())
		os.Exit(0)
	}

	level, err := log.ParseLevel(*config.Get().LogLevel)
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.SetLevel(level)
	log.SetReportCaller(true)

	if !*config.Get().LogPretty {
		log.SetFormatter(&log.JSONFormatter{
			TimestampFormat: time.RFC3339Nano,
		})
	}

	if err := config.Load(); err != nil {
		log.WithError(err).Debug()
	}

	hook, err := logrushooksentry.NewHook(logrushooksentry.Options{
		SentryDSN: *config.Get().SentryDSN,
		Release:   config.GetVersion(),
	})
	if err != nil {
		log.WithError(err).Fatal()
	}

	log.AddHook(hook)

	defer hook.Stop()

	log.Infof("Starting %s...", config.GetVersion())
	log.Debug(config.String())

	err = api.Init()
	if err != nil {
		log.WithError(err).Fatal()
	}

	err = queue.Init()
	if err != nil {
		log.WithError(err).Fatal()
	}

	if *config.Get().RedisEnabled {
		go queue.ScheduleMetrics()
	}

	// for redis
	queue.OnNewValue = func(message api.Message) {
		err := api.Send(message)
		if err != nil {
			log.
				WithError(err).
				WithField("message", message).
				Error("error in api.send")
			metrics.QueueErrorCounter.WithLabelValues(message.Type).Inc()

			message.RetryCount++
			message.RetryLastError = err.Error()

			_, err = queue.Add(message)

			if err != nil {
				log.
					WithError(err).
					WithField("message", message).
					Error("error in queue.Add")
			}
		}
	}

	web.StartServer()

	<-ctx.Done()
}
