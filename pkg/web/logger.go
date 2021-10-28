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
package web

import (
	"log"
	"net/http"
	"strings"

	logrus "github.com/sirupsen/logrus"
)

func logRequestHandler(server string, h http.Handler) http.Handler {
	logger := logrus.WithFields(logrus.Fields{
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

type httpServerLoggerAdapter struct{}

func (*httpServerLoggerAdapter) Write(p []byte) (n int, err error) {
	if message := string(p); strings.Contains(message, "TLS handshake error") {
		logrus.Debug(message)
	} else {
		logrus.Error(message)
	}

	return 0, nil
}

func httpServerLogger() *log.Logger {
	return log.New(&httpServerLoggerAdapter{}, "", 0)
}
