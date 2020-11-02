package main

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"
)

type Message struct {
	Type              string `json:"type"` // PUT,DELETE
	FileName          string `json:"fileName"`
	FileContent       string `json:"fileContent"`
	FileContentBase64 string `json:"fileContentBase64"`
	FilePermission    uint32 `json:"filePermission"` //defaults: 0644
	DirPermission     uint32 `json:"dirPermission"`  //defaults: 511
}

type Response struct {
	StatusCode int    `json:"statusCode"`
	StatusText string `json:"statusText"`
}

func makeDELETE(message Message) error {
	return os.Remove(message.FileName)
}

func makePUT(message Message) error {
	fileContent := []byte(message.FileContent)

	if len(message.FileContentBase64) > 0 {
		decoded, err := base64.StdEncoding.DecodeString(message.FileContentBase64)
		if err != nil {
			return err
		}

		fileContent = decoded
	}

	if message.FilePermission == 0 {
		message.FilePermission = uint32(0644)
	}

	if message.DirPermission == 0 {
		message.DirPermission = uint32(511)
	}

	fileDir := filepath.Dir(message.FileName)

	err := os.MkdirAll(fileDir, os.FileMode(message.DirPermission))
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(message.FileName, fileContent, os.FileMode(message.FilePermission))

	if err != nil {
		return err
	}

	return nil
}

func main() {
	http.HandleFunc("/api/endpoint", func(w http.ResponseWriter, r *http.Request) {
		message := Message{}
		err := json.NewDecoder(r.Body).Decode(&message)

		message.Type = strings.ToUpper(message.Type)
		if len(message.FileName) > 0 {
			message.FileName = path.Join("./data", message.FileName)
		}
		if err == nil {
			switch message.Type {
			case "PUT":
				err = makePUT(message)
			case "DELETE":
				err = makeDELETE(message)
			default:
				err = fmt.Errorf("unknown type %s", message.Type)
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
	})

	err := http.ListenAndServe(":9335", nil)
	if err != nil {
		panic(err)
	}
}
