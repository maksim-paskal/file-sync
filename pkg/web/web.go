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
package web

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"
	pprof "net/http/pprof"
	"strings"

	"github.com/maksim-paskal/file-sync/pkg/api"
	"github.com/maksim-paskal/file-sync/pkg/certs"
	"github.com/maksim-paskal/file-sync/pkg/config"
	"github.com/maksim-paskal/file-sync/pkg/metrics"
	"github.com/maksim-paskal/file-sync/pkg/queue"
	logrushooksentry "github.com/maksim-paskal/logrus-hook-sentry"
	log "github.com/sirupsen/logrus"
)

func StartServer() {
	go func() {
		_, serverCertBytes, _, serverKeyBytes, err := certs.NewCertificate("file-sync", certs.CertValidityMax)
		if err != nil {
			log.WithError(err).Fatal("failed to NewCertificate")
		}

		cert, err := tls.X509KeyPair(serverCertBytes, serverKeyBytes)
		if err != nil {
			log.WithError(err).Fatal("failed to NewCertificate")
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AddCert(certs.GetLoadedRootCert())

		server := &http.Server{
			Addr:    *config.Get().HTTPSAddress,
			Handler: logRequestHandler("sync", GetHTTPSRouter()),
			TLSConfig: &tls.Config{
				MinVersion:   tls.VersionTLS12,
				ClientAuth:   tls.RequireAndVerifyClientCert,
				ClientCAs:    caCertPool,
				Certificates: []tls.Certificate{cert},
			},
			ErrorLog: httpServerLogger(),
		}

		log.Infof("Start TLS server on %s", server.Addr)

		err = server.ListenAndServeTLS("", "")
		if err != nil {
			log.WithError(err).Fatal()
		}
	}()

	go func() {
		server := &http.Server{
			Addr:    *config.Get().HTTPAddress,
			Handler: logRequestHandler("queue", GetHTTPRouter()),
		}

		log.Infof("Start server on %s", server.Addr)

		err := server.ListenAndServe()
		if err != nil {
			log.WithError(err).Fatal()
		}
	}()

	go func() {
		server := &http.Server{
			Addr:    *config.Get().MetricsAddress,
			Handler: logRequestHandler("metrics", GetMetricsRouter()),
		}

		log.Infof("Start metrics server on %s", server.Addr)

		err := server.ListenAndServe()
		if err != nil {
			log.WithError(err).Fatal()
		}
	}()

	metrics.Up.Set(1)
}

func handlerSync(w http.ResponseWriter, r *http.Request) {
	message := api.Message{}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.
			WithError(err).
			WithFields(logrushooksentry.AddRequest(r)).
			WithField("message", message.String()).
			Error("error in ioutil.ReadAll")
	}

	err = json.Unmarshal(body, &message)
	if err != nil {
		log.
			WithError(err).
			WithFields(logrushooksentry.AddRequest(r)).
			WithField("message", message.String()).
			Error("error in json.Unmarshal")

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if log.GetLevel() <= log.DebugLevel {
		log.
			WithFields(logrushooksentry.AddRequest(r)).
			WithField("message", message.String()).
			Debug()
	}

	err = api.ProcessMessage(message)

	if err != nil {
		log.
			WithError(err).
			WithFields(logrushooksentry.AddRequest(r)).
			WithField("message", message.String()).
			Error("error in web.api.processMessage")
	}

	results := api.Response{
		Type:     message.Type,
		FileName: message.FileName,
	}

	if err != nil {
		results.StatusCode = 500
		results.StatusText = err.Error()
	} else {
		results.StatusCode = 200
		results.StatusText = "ok"
	}

	js, _ := json.Marshal(results)

	w.Header().Set("Content-Type", "application/json")

	_, err = w.Write(js)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handlerQueue(w http.ResponseWriter, r *http.Request) { //nolint:cyclop
	err := r.ParseForm()
	if err != nil {
		log.
			WithError(err).
			WithFields(logrushooksentry.AddRequest(r)).
			Error("error in r.ParseForm()")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		metrics.QueueErrorCounter.WithLabelValues("init").Inc()
	}

	value := r.Form.Get("value")
	debug := r.Form.Get("debug")
	force := r.Form.Get("force")

	if log.GetLevel() <= log.DebugLevel {
		log.WithFields(logrushooksentry.AddRequest(r)).Debug(value)
	}

	isDebugMode := len(debug) > 0 && strings.EqualFold(debug, "true")
	isForced := len(force) > 0 && strings.EqualFold(force, "true")

	if isDebugMode {
		log.WithFields(logrushooksentry.AddRequest(r)).Info("Debug mode")

		_, err = w.Write([]byte("ok"))
		if err != nil {
			log.
				WithError(err).
				WithFields(logrushooksentry.AddRequest(r)).
				Error("error in w.Write()")
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	message, err := api.GetMessageFromValue(value)

	if isForced {
		message.Force = true
	}

	if err != nil {
		log.
			WithError(err).
			WithFields(logrushooksentry.AddRequest(r)).
			WithField("message", message.String()).
			Error("error in web.api.getMessageFromValue")
		http.Error(w, err.Error(), http.StatusInternalServerError)
		metrics.QueueErrorCounter.WithLabelValues(message.Type).Inc()

		return
	}

	if log.GetLevel() <= log.DebugLevel {
		log.
			WithFields(logrushooksentry.AddRequest(r)).
			WithField("message", message.String()).
			Debug()
	}

	resultText := "ok"

	if *config.Get().RedisEnabled {
		id, err := queue.Add(message)
		if err != nil {
			log.
				WithError(err).
				WithFields(logrushooksentry.AddRequest(r)).
				WithField("message", message.String()).
				Error("error in web.queue.add")
			metrics.QueueErrorCounter.WithLabelValues(message.Type).Inc()

			return
		}

		resultText = id
	} else {
		go func() {
			err := api.SendWithRetry(message)
			if err != nil {
				log.
					WithError(err).
					WithFields(logrushooksentry.AddRequest(r)).
					WithField("message", message.String()).
					Error("error in web.api.send")
				metrics.QueueErrorCounter.WithLabelValues(message.Type).Inc()

				return
			}
		}()
	}

	metrics.QueueRequestCounter.WithLabelValues(message.Type).Inc()

	_, err = w.Write([]byte(resultText))
	if err != nil {
		log.
			WithError(err).
			WithFields(logrushooksentry.AddRequest(r)).
			WithField("message", message.String()).
			Error("error in w.Write")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handlerHealthz(w http.ResponseWriter, r *http.Request) {
	if _, err := w.Write([]byte("ok")); err != nil {
		log.
			WithError(err).
			WithFields(logrushooksentry.AddRequest(r)).
			Error("error in w.Write")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handlerQueueInfo(w http.ResponseWriter, r *http.Request) {
	meta, err := queue.Info()
	if err != nil {
		log.
			WithError(err).
			WithFields(logrushooksentry.AddRequest(r)).
			Error("error in queue.Info")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if _, err := w.Write([]byte(meta)); err != nil {
		log.
			WithError(err).
			WithFields(logrushooksentry.AddRequest(r)).
			Error("error in w.Write")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func handlerQueueFlush(w http.ResponseWriter, r *http.Request) {
	err := queue.Flush()
	if err != nil {
		log.
			WithError(err).
			WithFields(logrushooksentry.AddRequest(r)).
			Error("error in queue.Flush")
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if _, err := w.Write([]byte("ok")); err != nil {
		log.
			WithError(err).
			WithFields(logrushooksentry.AddRequest(r)).
			Error("error in w.Write")
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func GetHTTPRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/queue", handlerQueue)
	mux.HandleFunc("/api/queue/info", handlerQueueInfo)
	mux.HandleFunc("/api/queue/flush", handlerQueueFlush)
	mux.HandleFunc("/api/healthz", handlerHealthz)

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return mux
}

func GetHTTPSRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sync", handlerSync)
	mux.HandleFunc("/api/healthz", handlerHealthz)

	return mux
}

func GetMetricsRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.Handle("/metrics", metrics.GetHandler())

	return mux
}
