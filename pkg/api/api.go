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
package api

import (
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"io/ioutil"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/maksim-paskal/file-sync/pkg/certs"
	"github.com/maksim-paskal/file-sync/pkg/config"
	"github.com/maksim-paskal/file-sync/pkg/metrics"
	"github.com/maksim-paskal/file-sync/pkg/utils"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

const (
	MessageTypePut    = "put"
	MessageTypePatch  = "patch"
	MessageTypeDelete = "delete"
	MessageTypeCopy   = "copy"
	MessageTypeMove   = "move"
	defaultFileMode1  = fs.FileMode(0o777)
	defaultFileMode2  = fs.FileMode(0o600)
	defaultFileMode3  = fs.FileMode(0o644)
	maxRetryCount     = 20
)

type Message struct {
	ID                string `json:"id"`
	RetryCount        int    `json:"retryCount"`
	RetryLastError    string `json:"retryLastError"`
	Type              string `json:"type"`
	FileName          string `json:"fileName"`
	NewFileName       string `json:"newFileName"`
	Force             bool   `json:"force"`
	FileContent       string `json:"fileContent"`
	FileContentBase64 string `json:"fileContentBase64"`
	SHA256            string `json:"sha256"`
}

type Response struct {
	Type       string `json:"type"`
	FileName   string `json:"fileName"`
	StatusCode int    `json:"statusCode"`
	StatusText string `json:"statusText"`
}

var client *http.Client

func Init() error {
	_, serverCertBytes, _, serverKeyBytes, err := certs.NewCertificate("file-sync", certs.CertValidityMax)
	if err != nil {
		return errors.Wrap(err, "failed to NewCertificate")
	}

	cert, err := tls.X509KeyPair(serverCertBytes, serverKeyBytes)
	if err != nil {
		return errors.Wrap(err, "failed to NewCertificate")
	}

	caCertPool := x509.NewCertPool()
	caCertPool.AddCert(certs.GetLoadedRootCert())

	// Setup HTTPS client
	tlsConfig := &tls.Config{
		Certificates:       []tls.Certificate{cert},
		RootCAs:            caCertPool,
		InsecureSkipVerify: true, //nolint:gosec
	}

	tlsConfig.BuildNameToCertificate()

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	client = &http.Client{
		Transport: transport,
		Timeout:   *config.Get().SyncTimeout,
	}

	return nil
}

func makeCopy(message Message) error {
	message.FileName = path.Join(*config.Get().DestinationDir, message.FileName)
	message.NewFileName = path.Join(*config.Get().DestinationDir, message.NewFileName)

	sourceFileStat, err := os.Stat(message.FileName)
	if err != nil {
		return errors.Wrap(err, "error in os.Stat")
	}

	if !sourceFileStat.Mode().IsRegular() {
		return fmt.Errorf("%s is not a regular file", message.FileName)
	}

	source, err := os.Open(message.FileName)
	if err != nil {
		return errors.Wrap(err, "error in os.Open")
	}
	defer source.Close()

	destination, err := os.Create(message.NewFileName)
	if err != nil {
		return errors.Wrap(err, "error in os.Create")
	}
	defer destination.Close()
	_, err = io.Copy(destination, source)

	return errors.Wrap(err, "error in io.Copy")
}

func makeMove(message Message) error {
	message.FileName = path.Join(*config.Get().DestinationDir, message.FileName)
	message.NewFileName = path.Join(*config.Get().DestinationDir, message.NewFileName)

	if _, err := os.Stat(message.FileName); os.IsNotExist(err) {
		// if file not found and reach maxRetryCount exit with success, logs warning
		if message.RetryCount > maxRetryCount {
			metrics.QueueMaxRetryCountCounter.WithLabelValues(MessageTypeMove).Inc()
			log.
				WithError(ErrFileNotFound).
				WithField("message", message).
				Warn()

			return nil
		}

		return errors.Wrapf(err, "%s not exists", message.FileName)
	}

	fileDir := filepath.Dir(message.NewFileName)

	err := os.MkdirAll(fileDir, defaultFileMode1)
	if err != nil {
		return errors.Wrap(err, "error in os.MkdirAll")
	}

	err = os.Rename(message.FileName, message.NewFileName)
	if err != nil {
		return errors.Wrap(err, "error in os.Rename")
	}

	log.Infof("%s file %s", message.Type, message.FileName)

	return nil
}

func makeDelete(message Message) error {
	message.FileName = path.Join(*config.Get().DestinationDir, message.FileName)

	if _, err := os.Stat(message.FileName); os.IsNotExist(err) {
		// if file not found and reach maxRetryCount exit with success, logs warning
		if message.RetryCount > maxRetryCount {
			metrics.QueueMaxRetryCountCounter.WithLabelValues(MessageTypeDelete).Inc()
			log.
				WithError(ErrFileNotFound).
				WithField("message", message).
				Warn()

			return nil
		}

		return errors.Wrapf(err, "%s not exists", message.FileName)
	}

	err := os.Remove(message.FileName)
	if err != nil {
		return errors.Wrap(err, "error in os.Remove")
	}

	log.Infof("%s file %s", message.Type, message.FileName)

	return nil
}

func makeSave(message Message) error { //nolint:cyclop
	message.FileName = path.Join(*config.Get().DestinationDir, message.FileName)

	fileInfo, err := os.Stat(message.FileName)
	isFileNameNotExists := os.IsNotExist(err)

	if err == nil && fileInfo.IsDir() {
		return fmt.Errorf("%s is directory", message.FileName)
	}

	log.Debugf("FileName=%s,isFileNameNotExists=%t", message.FileName, isFileNameNotExists)

	switch message.Type {
	case MessageTypePut:
		if !isFileNameNotExists {
			if message.Force {
				log.
					WithError(ErrFileMustNotExists).
					WithField("message", message).
					Warn()
			} else {
				return ErrFileMustNotExists
			}
		}
	case MessageTypePatch:
		if isFileNameNotExists {
			if message.Force {
				log.
					WithError(ErrFileMustExists).
					WithField("message", message).
					Warn()
			} else {
				return ErrFileMustExists
			}
		}
	}

	fileContent := []byte(message.FileContent)

	if len(message.FileContentBase64) > 0 {
		decoded, err := base64.StdEncoding.DecodeString(message.FileContentBase64)
		if err != nil {
			return errors.Wrap(err, "error in base64.StdEncoding.DecodeString")
		}

		fileContent = decoded
	}

	fileDir := filepath.Dir(message.FileName)

	err = os.MkdirAll(fileDir, defaultFileMode1)
	if err != nil {
		return errors.Wrap(err, "error in os.MkdirAll")
	}

	err = ioutil.WriteFile(message.FileName, fileContent, defaultFileMode2)
	if err != nil {
		return errors.Wrap(err, "error in ioutil.WriteFile")
	}

	err = os.Chmod(message.FileName, defaultFileMode3)
	if err != nil {
		return errors.Wrap(err, "error in os.Chmod")
	}

	if len(message.SHA256) != 0 {
		data, err := ioutil.ReadFile(message.FileName)
		if err != nil {
			return errors.Wrap(err, "error in ioutil.ReadFile")
		}

		if message.SHA256 != utils.NewSHA256(data) {
			log.
				WithError(ErrSHA256Failed).
				WithField("message", message).
				Warn()
		}
	}

	log.Infof("%s file %s", message.Type, message.FileName)

	return nil
}

func Send(message Message) error {
	ctx := context.Background()

	jsonStr, err := json.Marshal(message)
	if err != nil {
		return errors.Wrap(err, "error in json.Marshal")
	}

	url := fmt.Sprintf("https://%s/api/sync", *config.Get().SyncAddress)

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonStr))
	if err != nil {
		return errors.Wrap(err, "error in http.NewRequestWithContext")
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}

	if resp == nil {
		return errors.Wrap(err, "error in client.Do")
	}

	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return errors.New("status != 200")
	}

	// Dump response
	data, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return errors.Wrap(err, "error in ioutil.ReadAll")
	}

	log.Debugf("url=%s,result=%s", url, string(data))

	results := Response{}

	err = json.Unmarshal(data, &results)
	if err != nil {
		return errors.Wrap(err, "error in json.Unmarshal")
	}

	if results.StatusCode != http.StatusOK {
		return errors.New(results.StatusText)
	}

	return nil
}

func GetMessageFromValue(value string) (Message, error) {
	message := Message{}

	if len(value) == 0 {
		return message, errors.New("no value")
	}

	matched, err := regexp.Match(`^(put|patch|delete|copy|move):.+$`, []byte(value))
	if err != nil {
		return message, errors.Wrap(err, "error in regexp.Match")
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
		filePath := path.Join(*config.Get().SourceDir, dataValues[1])
		fileInfo, err := os.Stat(filePath)

		if os.IsNotExist(err) {
			return message, fmt.Errorf("file %s not found", filePath)
		}

		if fileInfo.IsDir() {
			return message, fmt.Errorf("file %s is directory", filePath)
		}

		fileContent, err := ioutil.ReadFile(filePath)
		if err != nil {
			return message, errors.Wrap(err, "error in ioutil.ReadFile")
		}

		message.SHA256 = utils.NewSHA256(fileContent)
		message.FileContentBase64 = base64.StdEncoding.EncodeToString(fileContent)
	}

	return message, nil
}

func ProcessMessage(message Message) error {
	switch message.Type {
	case MessageTypePut:
		return makeSave(message)
	case MessageTypePatch:
		return makeSave(message)
	case MessageTypeDelete:
		return makeDelete(message)
	case MessageTypeCopy:
		return makeCopy(message)
	case MessageTypeMove:
		return makeMove(message)
	default:
		return fmt.Errorf("unknown type %s", message.Type)
	}
}
