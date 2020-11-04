package main

import (
	"encoding/base64"
	"io/ioutil"
	"os"
	"path/filepath"
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

type API struct {
}

func newAPI() *API {
	api := API{}

	return &api
}

func (api *API) makeDELETE(message Message) error {
	return os.Remove(message.FileName)
}

func (api *API) makePUT(message Message) error {
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
