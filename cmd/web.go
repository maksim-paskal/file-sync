package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"io/ioutil"
	"net/http"
	"path"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Web struct {
	api    *API
	queue  *Queue
	config *Config
}

func newWeb(config *Config, queue *Queue) *Web {
	web := Web{
		api:    newAPI(config),
		queue:  queue,
		config: config,
	}

	go func() {
		caCertPEM, err := ioutil.ReadFile("ssl/ca.crt")
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
			Addr:    ":9335",
			Handler: mux,
			TLSConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
				ClientAuth: tls.RequireAndVerifyClientCert,
				ClientCAs:  roots,
			},
		}

		log.Info("Start TLS server on :9335")

		err = server.ListenAndServeTLS("ssl/server.crt", "ssl/server.key")
		if err != nil {
			log.Panic(err)
		}
	}()

	go func() {
		mux := http.NewServeMux()
		mux.HandleFunc("/api/queue", web.handlerQueue)

		server := &http.Server{
			Addr:    ":9336",
			Handler: mux,
		}

		log.Info("Start server on :9336")

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

	message.Type = strings.ToUpper(message.Type)
	if len(message.FileName) > 0 {
		message.FileName = path.Join("./data", message.FileName)
	}

	if err == nil {
		switch message.Type {
		case "PUT":
			err = web.api.makePUT(message)
		case "DELETE":
			err = web.api.makeDELETE(message)
		default:
			err = ErrUnknownType
		}
	}

	results := Response{}

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

	data := r.Form.Get("value")

	if len(data) == 0 {
		http.Error(w, "no value", http.StatusBadRequest)

		return
	}

	matched, err := regexp.Match(`(add|patch|delete):.+`, []byte(data))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if !matched {
		http.Error(w, "invalid format", http.StatusInternalServerError)

		return
	}

	web.queue.add(data)

	_, err = w.Write([]byte("ok"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
