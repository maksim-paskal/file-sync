package main

import (
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	log "github.com/sirupsen/logrus"
)

type Web struct {
	api *API
}

func newWeb() *Web {
	web := Web{
		api: newAPI(),
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
			Handler: web.getHTTPSRouter(),
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
			Handler: web.getHTTPRouter(),
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

	switch message.Type {
	case MessageTypePut:
		err = web.api.makeSave(message)
	case MessageTypePatch:
		err = web.api.makeSave(message)
	case MessageTypeDelete:
		err = web.api.makeDelete(message)
	default:
		err = fmt.Errorf("unknown type %s", message.Type)
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
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}

	value := r.Form.Get("value")
	debug := r.Form.Get("debug")

	if log.GetLevel() <= log.DebugLevel {
		log.Debug(value)
	}

	message, err := web.api.getMessageFromValue(value)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)

		return
	}

	if log.GetLevel() <= log.DebugLevel {
		r, _ := json.Marshal(message)
		log.Debug(string(r))
	}

	if len(debug) > 0 && strings.EqualFold(debug, "true") {
		log.Info("Debug mode")
	} else {
		go func() {
			err := web.api.send(message)
			if err != nil {
				log.Error(err)
			}
		}()
	}

	_, err = w.Write([]byte("ok"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (web *Web) getHTTPRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/queue", web.handlerQueue)

	return mux
}

func (web *Web) getHTTPSRouter() *http.ServeMux {
	mux := http.NewServeMux()
	mux.HandleFunc("/api/sync", web.handlerSync)

	return mux
}
