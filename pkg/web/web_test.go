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
package web_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maksim-paskal/file-sync/pkg/api"
	"github.com/maksim-paskal/file-sync/pkg/certs"
	"github.com/maksim-paskal/file-sync/pkg/config"
	"github.com/maksim-paskal/file-sync/pkg/web"
)

var ctx = context.Background()

func init() { //nolint: gochecknoinits
	if err := config.Load(); err != nil {
		panic(err)
	}

	if err := certs.Init(); err != nil {
		panic(err)
	}

	if err := api.Init(); err != nil {
		panic(err)
	}

	web.Init()
}

func TestRouting_Queue(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(web.GetHTTPRouter())
	defer srv.Close()

	queueURL := fmt.Sprintf("%s/api/queue?value=put:tests/test.txt", srv.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queueURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if res.StatusCode == http.StatusOK {
		t.Error("must be error")
	}

	stringBody := string(body)

	hosts := web.GetSyncAddress()
	if len(hosts) != 3 {
		t.Error("must be 3 hosts")
	}

	for _, host := range hosts {
		if !strings.Contains(stringBody, host) {
			t.Error("text must contain 10.10.10.10", stringBody)
		}
	}
}

func TestRouting_Sync(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(web.GetHTTPSRouter())
	defer srv.Close()

	queueURL := fmt.Sprintf("%s/api/sync", srv.URL)

	message := api.Message{
		Type:              "put",
		FileName:          "tests/test-http.txt",
		FileContentBase64: "ZHNkZA==",
		SHA256:            "701df70cc797a5d18f69fbf8fa538b15c5adcc06e51de80b446d465696d6c3b5",
	}

	jsonStr, err := json.Marshal(message)
	if err != nil {
		t.Error(err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, queueURL, bytes.NewBuffer(jsonStr))
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != `{"type":"put","fileName":"tests/test-http.txt","statusCode":200,"statusText":"ok"}` {
		t.Fatalf("text %s not OK", string(body))
	}

	if res.StatusCode != http.StatusOK {
		t.Fatalf("status %d not OK", res.StatusCode)
	}
}

func TestRouting_Healthz(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(web.GetHTTPRouter())
	defer srv.Close()

	queueURL := fmt.Sprintf("%s/api/healthz", srv.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queueURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if string(body) != "ok" {
		t.Errorf("text %s not OK", string(body))
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("status %d not OK", res.StatusCode)
	}
}

func TestRouting_Metrics(t *testing.T) {
	t.Parallel()

	srv := httptest.NewServer(web.GetMetricsRouter())
	defer srv.Close()

	queueURL := fmt.Sprintf("%s/metrics", srv.URL)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, queueURL, nil)
	if err != nil {
		t.Fatal(err)
	}

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	defer res.Body.Close()

	body, err := ioutil.ReadAll(res.Body)
	if err != nil {
		t.Fatal(err)
	}

	if m := "filesync_up"; !strings.Contains(string(body), m) {
		t.Fatal(fmt.Sprintf("no metric %s found", m))
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("status %d not OK", res.StatusCode)
	}
}
