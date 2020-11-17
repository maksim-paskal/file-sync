package main

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	log "github.com/sirupsen/logrus"
)

const (
	MessageTypePut    = "put"
	MessageTypePatch  = "patch"
	MessageTypeDelete = "delete"
	MessageTypeCopy   = "copy"
	MessageTypeMove   = "move"
)

type Message struct {
	Type              string `json:"type"`
	FileName          string `json:"fileName"`
	NewFileName       string `json:"newFileName"`
	Force             bool   `json:"force"`
	FileContent       string `json:"fileContent"`
	FileContentBase64 string `json:"fileContentBase64"`
	SHA256            string `json:"SHA256"`
}

type Response struct {
	Type       string `json:"type"`
	FileName   string `json:"fileName"`
	StatusCode int    `json:"statusCode"`
	StatusText string `json:"statusText"`
}

type API struct {
}

func newAPI() *API {
	api := API{}

	return &api
}

func (api *API) makeCopy(message Message) error {
	message.FileName = path.Join(*appConfig.destinationDir, message.FileName)
	message.NewFileName = path.Join(*appConfig.destinationDir, message.NewFileName)

	sourceFileStat, err := os.Stat(message.FileName)
	if err != nil {
		return err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", message.FileName)
	}

	source, err := os.Open(message.FileName)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(message.NewFileName)
	if err != nil {
		return err
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)

	return err
}

func (api *API) makeMove(message Message) error {
	message.FileName = path.Join(*appConfig.destinationDir, message.FileName)
	message.NewFileName = path.Join(*appConfig.destinationDir, message.NewFileName)

	log.Info(message.FileName)
	log.Info(message.NewFileName)

	err := os.Rename(message.FileName, message.NewFileName)
	if err != nil {
		log.Fatal(err)
	}

	log.Infof("%s file %s", message.Type, message.FileName)

	return nil
}

func (api *API) makeDelete(message Message) error {
	message.FileName = path.Join(*appConfig.destinationDir, message.FileName)

	err := os.Remove(message.FileName)
	if err != nil {
		return err
	}

	log.Infof("%s file %s", message.Type, message.FileName)

	return nil
}

func (api *API) makeSave(message Message) error {
	message.FileName = path.Join(*appConfig.destinationDir, message.FileName)

	fileInfo, err := os.Stat(message.FileName)
	isFileNameNotExists := os.IsNotExist(err)

	if err == nil && fileInfo.IsDir() {
		return fmt.Errorf("%s is directory", message.FileName)
	}

	log.Debugf("FileName=%s,isFileNameNotExists=%t", message.FileName, isFileNameNotExists)

	switch message.Type {
	case MessageTypePut:
		if !isFileNameNotExists {
			err := fmt.Errorf("file %s must not exists", message.FileName)

			if message.Force {
				log.Error(err)
			} else {
				return err
			}
		}
	case MessageTypePatch:
		if isFileNameNotExists {
			err := fmt.Errorf("file %s must exists", message.FileName)

			if message.Force {
				log.Error(err)
			} else {
				return err
			}
		}
	}

	fileContent := []byte(message.FileContent)

	if len(message.FileContentBase64) > 0 {
		decoded, err := base64.StdEncoding.DecodeString(message.FileContentBase64)
		if err != nil {
			return err
		}

		fileContent = decoded
	}

	fileDir := filepath.Dir(message.FileName)

	err = os.MkdirAll(fileDir, 0777)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(message.FileName, fileContent, 0600)
	if err != nil {
		return err
	}

	err = os.Chmod(message.FileName, 0664)
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

	log.Infof("%s file %s", message.Type, message.FileName)

	return nil
}

func (api *API) send(message Message) error {
	ctx := context.Background()

	// Load client cert
	cert, err := tls.LoadX509KeyPair(*appConfig.sslClientCrt, *appConfig.sslClientKey)
	if err != nil {
		return err
	}

	// Load CA cert
	caCert, err := ioutil.ReadFile(*appConfig.sslCA)
	if err != nil {
		return err
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AppendCertsFromPEM(caCert)

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, //nolint:gosec
	}

	tlsConfig.BuildNameToCertificate()

	transport := &http.Transport{TLSClientConfig: tlsConfig}
	client := &http.Client{
		Transport: transport,
		Timeout:   *appConfig.syncTimeout,
	}

	jsonStr, err := json.Marshal(message)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("https://%s/api/sync", *appConfig.syncAddress)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonStr))
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

	log.Debugf("url=%s,result=%s", url, string(data))

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

func (api *API) getMessageFromValue(value string) (Message, error) {
	message := Message{}

	if len(value) == 0 {
		return message, errors.New("no value")
	}

	matched, err := regexp.Match(`^(put|patch|delete|copy|move):.+$`, []byte(value))
	if err != nil {
		return message, err
	}

	if !matched {
		return message, errors.New("value not correct")
	}

	dataValues := strings.Split(value, ":")

	message.Type = dataValues[0]
	message.FileName = dataValues[1]
	dataValuesLen := len(dataValues)

	if dataValuesLen > 2 { //nolint:gomnd
		message.NewFileName = dataValues[2]
	}

	isSrcOperations := message.Type == MessageTypePut || message.Type == MessageTypePatch

	if isSrcOperations {
		filePath := path.Join(*appConfig.sourceDir, dataValues[1])
		fileInfo, err := os.Stat(filePath)

		if os.IsNotExist(err) {
			return message, fmt.Errorf("file %s not found", filePath)
		}

		if fileInfo.IsDir() {
			return message, fmt.Errorf("file %s is directory", filePath)
		}

		fileContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			return message, err
		}

		message.SHA256 = NewSHA256(fileContent)
		message.FileContentBase64 = base64.StdEncoding.EncodeToString(fileContent)
	}

	return message, nil
}

func (api *API) processMessage(message Message) error {
	switch message.Type {
	case MessageTypePut:
		return api.makeSave(message)
	case MessageTypePatch:
		return api.makeSave(message)
	case MessageTypeDelete:
		return api.makeDelete(message)
	case MessageTypeCopy:
		return api.makeCopy(message)
	case MessageTypeMove:
		return api.makeMove(message)
	default:
		return fmt.Errorf("unknown type %s", message.Type)
	}
}
