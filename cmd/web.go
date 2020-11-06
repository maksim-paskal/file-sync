package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Web struct {
	api    *API
	config *Config
}

func newWeb(config *Config) *Web {
	web := Web{
		api:    newAPI(config),
		config: config,
	}

	go func() {
		caCertPEM, err := ioutil.ReadFile(*config.sslCA)
		if err != nil {
			log.Panic("can not load ca")
		}

		roots := x509.NewCertPool()
		ok := roots.AppendCertsFromPEM(caCertPEM)

		if !ok {
			log.Panic("failed to parse root certificate")
		}

		mux := http.NewServeMux()
		mux.HandleFunc("/api/sync", web.handlerSync)

		server := &http.Server{
			Addr:    *config.httpsAddress,
			Handler: mux,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				ClientAuth: tls.RequireAndVerifyClientCert,
				ClientCAs:  roots,
			},
		}

		log.Infof("Start TLS server on %s", server.Addr)

		err = server.ListenAndServeTLS(*config.sslServerCrt, *config.sslServerKey)
		if err != nil {
			log.Panic(err)
		}
	}()

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/queue", web.handlerQueue)

		server := &http.Server{
			Addr:    *config.httpAddress,
			Handler: mux,
		}

		log.Infof("Start server on %s", server.Addr)

		err := server.ListenAndServe()
		if err != nil {
			log.Panic(err)
		}
	}()

	return &web
}

func (web *Web) handlerSync(w http.ResponseWriter, r *http.Request) {
	message := Message{}
	err := json.NewDecoder(r.Body).Decode(&message)

	if log.GetLevel() <= log.DebugLevel {
		r, _ := json.Marshal(message)
		log.Debug(string(r))
	}

	message.Type = strings.ToUpper(message.Type)
	if len(message.FileName) > 0 {
		message.FileName = path.Join(*web.config.destinationDir, message.FileName)
	}

	if err == nil {
		switch message.Type {
		case "PUT":
			err = web.api.makePUT(message)
		case "DELETE":
			err = web.api.makeDELETE(message)
		default:
			err = fmt.Errorf("unknown type %s", message.Type)
		}
	}

	results := Response{
		FileName: message.FileName,
	}

	if err != nil {
		results.StatusCode = 500
		results.StatusText = err.Error()

		w.WriteHeader(http.StatusInternalServerError)
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
	}

	value := r.Form.Get("value")

	message, err := web.api.getMessageFromValue(value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	go func() {
		err := web.api.send(message)
		if err != nil {
			log.Error(err)
		}
	}()

	if log.GetLevel() <= log.DebugLevel {
		r, _ := json.Marshal(message)
		log.Debug(string(r))
	}

	_, err = w.Write([]byte("ok"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
