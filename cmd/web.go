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
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	pprof "net/http/pprof"
	"strings"

	logrushooksentry "github.com/maksim-paskal/logrus-hook-sentry"
	log "github.com/sirupsen/logrus"
)

type Web struct {
	api      *API
	exporter *Exporter
	queue    *Queue
}

func newWeb(exporter *Exporter, queue *Queue, api *API) *Web {
	web := Web{
		api:      api,
		exporter: exporter,
		queue:    queue,
	}

	return &web
}

func (web *Web) startServer() {
	go func() {
		caCertPEM, err := ioutil.ReadFile(*appConfig.sslCA)
		if err != nil {
			log.WithError(err).Fatal("can not load ca")
		}

		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM(caCertPEM)

		if !ok {
			log.Fatal("failed to parse root certificate")
		}

		server := &http.Server{
			Addr:    *appConfig.httpsAddress,
			Handler: web.logRequestHandler("sync", web.getHTTPSRouter()),
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				ClientAuth: tls.RequireAndVerifyClientCert,
				ClientCAs:  roots,
			},
		}

		log.Infof("Start TLS server on %s", server.Addr)

		err = server.ListenAndServeTLS(*appConfig.sslServerCrt, *appConfig.sslServerKey)
		if err != nil {
			log.WithError(err).Fatal()
		}
	}()

	go func() {
		server := &http.Server{
			Addr:    *appConfig.httpAddress,
			Handler: web.logRequestHandler("queue", web.getHTTPRouter()),
		}

		log.Infof("Start server on %s", server.Addr)

		err := server.ListenAndServe()
		if err != nil {
			log.WithError(err).Fatal()
		}
	}()
}

func (web *Web) handlerSync(w http.ResponseWriter, r *http.Request) {
	message := Message{}

	defer r.Body.Close()

	body, err := ioutil.ReadAll(r.Body)
	if err != nil {
		log.
			WithError(err).
			WithField(logrushooksentry.RequestKey, r).
			WithField("message", message).
			Error()
	}

	err = json.Unmarshal(body, &message)
	if err != nil {
		log.
			WithError(err).
			WithField(logrushooksentry.RequestKey, r).
			WithField("message", message).
			Error()

		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if log.GetLevel() <= log.DebugLevel {
		r, _ := json.Marshal(message)
		log.WithField(logrushooksentry.RequestKey, r).Debug(string(r))
	}

	err = web.api.processMessage(message)

	if err != nil {
		log.
			WithError(err).
			WithField(logrushooksentry.RequestKey, r).
			WithField("message", message).
			Error()
	}

	results := Response{
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

func (web *Web) handlerQueue(w http.ResponseWriter, r *http.Request) {
	err := r.ParseForm()
	if err != nil {
		log.
			WithError(err).
			WithField(logrushooksentry.RequestKey, r).
			Error()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		web.exporter.queueErrorCounter.WithLabelValues("init").Inc()
	}

	value := r.Form.Get("value")
	debug := r.Form.Get("debug")
	force := r.Form.Get("force")

	if log.GetLevel() <= log.DebugLevel {
		log.WithField(logrushooksentry.RequestKey, r).Debug(value)
	}

	isDebugMode := len(debug) > 0 && strings.EqualFold(debug, "true")
	isForced := len(force) > 0 && strings.EqualFold(force, "true")

	if isDebugMode {
		log.WithField(logrushooksentry.RequestKey, r).Info("Debug mode")

		_, err = w.Write([]byte("ok"))
		if err != nil {
			log.
				WithError(err).
				WithField(logrushooksentry.RequestKey, r).
				Error()
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	message, err := web.api.getMessageFromValue(value)

	if isForced {
		message.Force = true
	}

	if err != nil {
		log.
			WithError(err).
			WithField(logrushooksentry.RequestKey, r).
			WithField("message", message).
			Error()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		web.exporter.queueErrorCounter.WithLabelValues(message.Type).Inc()

		return
	}

	if log.GetLevel() <= log.DebugLevel {
		r, _ := json.Marshal(message)
		log.
			WithField(logrushooksentry.RequestKey, r).
			WithField("message", message).
			Debug(string(r))
	}

	resultText := "ok"

	if *appConfig.redisEnabled {
		id, err := web.queue.add(message)
		if err != nil {
			log.
				WithError(err).
				WithField(logrushooksentry.RequestKey, r).
				WithField("message", message).
				Error()
			web.exporter.queueErrorCounter.WithLabelValues(message.Type).Inc()

			return
		}

		resultText = fmt.Sprintf("total queue size = %d", id)
	} else {
		go func() {
			err := web.api.send(message)
			if err != nil {
				log.
					WithError(err).
					WithField(logrushooksentry.RequestKey, r).
					WithField("message", message).
					Error()
				web.exporter.queueErrorCounter.WithLabelValues(message.Type).Inc()

				return
			}
		}()
	}

	web.exporter.queueRequestCounter.WithLabelValues(message.Type).Inc()

	_, err = w.Write([]byte(resultText))
	if err != nil {
		log.
			WithError(err).
			WithField(logrushooksentry.RequestKey, r).
			WithField("message", message).
			Error()
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (web *Web) handlerHealthz(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("ok"))
	if err != nil {
		log.
			WithError(err).
			WithField(logrushooksentry.RequestKey, r).
			Error()
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (web *Web) getHTTPRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/queue", web.handlerQueue)
	mux.HandleFunc("/api/healthz", web.handlerHealthz)

	mux.HandleFunc("/debug/pprof/", pprof.Index)
	mux.HandleFunc("/debug/pprof/cmdline", pprof.Cmdline)
	mux.HandleFunc("/debug/pprof/profile", pprof.Profile)
	mux.HandleFunc("/debug/pprof/symbol", pprof.Symbol)
	mux.HandleFunc("/debug/pprof/trace", pprof.Trace)

	return mux
}

func (web *Web) getHTTPSRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sync", web.handlerSync)

	return mux
}

func (web *Web) logRequestHandler(server string, h http.Handler) http.Handler {
	logger := log.WithFields(log.Fields{
		"server": server,
	})

	fn := func(w http.ResponseWriter, r *http.Request) {
		h.ServeHTTP(w, r)

		if r.URL.Path == "/api/healthz" {
			logger.Debugf("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		} else {
			logger.Infof("%s %s %s", r.RemoteAddr, r.Method, r.URL)
		}
	}

	return http.HandlerFunc(fn)
}
