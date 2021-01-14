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
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRouting_Queue(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	api, err := newAPI()
	if err != nil {
		t.Fatal(err)
	}

	exporter := newExporter()
	queue := newQueue("file-sync")
	web := newWeb(exporter, queue, api)

	srv := httptest.NewServer(web.getHTTPRouter())
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

	if string(body) != "ok" {
		t.Errorf("text %s not OK", string(body))
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("status %d not OK", res.StatusCode)
	}
}

func TestRouting_Sync(t *testing.T) {
	t.Parallel()

	ctx := context.Background()

	api, err := newAPI()
	if err != nil {
		t.Fatal(err)
	}

	exporter := newExporter()
	queue := newQueue("file-sync")
	web := newWeb(exporter, queue, api)

	srv := httptest.NewServer(web.getHTTPSRouter())
	defer srv.Close()

	queueURL := fmt.Sprintf("%s/api/sync", srv.URL)

	message := Message{
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
		t.Errorf("text %s not OK", string(body))
	}

	if res.StatusCode != http.StatusOK {
		t.Errorf("status %d not OK", res.StatusCode)
	}
}
