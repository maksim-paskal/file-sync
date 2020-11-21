package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Web struct {
	api      *API
	exporter *Exporter
}

func newWeb(exporter *Exporter) *Web {
	web := Web{
		api:      newAPI(),
		exporter: exporter,
	}

	return &web
}

func (web *Web) startServer() {
	go func() {
		caCertPEM, err := ioutil.ReadFile(*appConfig.sslCA)
		if err != nil {
			log.Panic("can not load ca")
		}

		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM(caCertPEM)

		if !ok {
			log.Panic("failed to parse root certificate")
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
			log.Panic(err)
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
			log.Panic(err)
		}
	}()
}

func (web *Web) handlerSync(w http.ResponseWriter, r *http.Request) {
	message := Message{}

	err := json.NewDecoder(r.Body).Decode(&message)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if log.GetLevel() <= log.DebugLevel {
		r, _ := json.Marshal(message)
		log.Debug(string(r))
	}

	err = web.api.processMessage(message)

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
		http.Error(w, err.Error(), http.StatusInternalServerError)
		web.exporter.queueErr.Inc()
	}

	value := r.Form.Get("value")
	debug := r.Form.Get("debug")
	force := r.Form.Get("force")

	if log.GetLevel() <= log.DebugLevel {
		log.Debug(value)
	}

	isDebugMode := len(debug) > 0 && strings.EqualFold(debug, "true")
	isForced := len(force) > 0 && strings.EqualFold(force, "true")

	if isDebugMode {
		log.Info("Debug mode")

		_, err = w.Write([]byte("ok"))
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}

		return
	}

	message, err := web.api.getMessageFromValue(value)

	if isForced {
		message.Force = true
	}

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		web.exporter.queueErr.Inc()

		return
	}

	if log.GetLevel() <= log.DebugLevel {
		r, _ := json.Marshal(message)
		log.Debug(string(r))
	}

	go func() {
		err := web.api.send(message)
		if err != nil {
			log.Error(err)
		}
	}()

	web.exporter.queueAdd.Inc()

	_, err = w.Write([]byte("ok"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (web *Web) handlerHealthz(w http.ResponseWriter, r *http.Request) {
	_, err := w.Write([]byte("ok"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (web *Web) getHTTPRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/queue", web.handlerQueue)
	mux.HandleFunc("/api/healthz", web.handlerHealthz)

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
