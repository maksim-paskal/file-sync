package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	log "github.com/sirupsen/logrus"
)

type Message struct {
	Type              string `json:"type"` // PUT,DELETE
	FileName          string `json:"fileName"`
	FileContent       string `json:"fileContent"`
	FileContentBase64 string `json:"fileContentBase64"`
	SHA256            string `json:"SHA256"`
}

type Response struct {
	StatusCode int    `json:"statusCode"`
	StatusText string `json:"statusText"`
}

type API struct {
	config *Config
}

func newAPI(config *Config) *API {
	api := API{
		config: config,
	}

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

	fileDir := filepath.Dir(message.FileName)

	err := os.MkdirAll(fileDir, os.FileMode(511))
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(message.FileName, fileContent, os.FileMode(0644))

	if err != nil {
		return err
	}

	if len(message.SHA256) != 0 {
		data, err := ioutil.ReadFile(message.FileName)
		if err != nil {
			return err
		}

		if message.SHA256 != NewSHA256(data) {
			log.Warnf("file %s SHA256 check failed", message.FileName)
		}
	}

	return nil
}

func (api *API) send(message Message) error {
	ctx := context.Background()

	// Load client cert
	cert, err := tls.LoadX509KeyPair("ssl/client01.crt", "ssl/client01.key")
	if err != nil {
		return err
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile("ssl/ca.crt")
	if err != nil {
		return err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true,
	}

	tlsConfig.BuildNameToCertificate()

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{Transport: transport}

	jsonStr, err := json.Marshal(message)
	if err != nil {
		return err
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://localhost:9335/api/sync", bytes.NewBuffer(jsonStr))
	if err != nil {
		return err
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("status != 200")
	}

	// Dump response
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	log.Infof("result=%s", string(data))

	results := Response{}

	err = json.Unmarshal(data, &results)
	if err != nil {
		return err
	}

	if results.StatusCode != http.StatusOK {
		return errors.New(results.StatusText)
	}

	return nil
}
