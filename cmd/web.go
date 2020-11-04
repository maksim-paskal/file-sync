package main

import (
	"encoding/json"
	"net/http"
	"path"
	"strings"
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

	http.HandleFunc("/api/endpoint", web.handlerEndpoint)
	http.HandleFunc("/api/queue", web.handlerQueue)

	go func() {
		err := http.ListenAndServe(":9335", nil)
		if err != nil {
			panic(err)
		}
	}()

	return &web
}

func (web *Web) handlerEndpoint(w http.ResponseWriter, r *http.Request) {
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
		http.Error(w, "no value", http.StatusInternalServerError)

		return
	}

	web.queue.add(data)

	_, err = w.Write([]byte("ok"))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}
