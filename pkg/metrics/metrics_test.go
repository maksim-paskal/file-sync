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
package metrics_test

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/maksim-paskal/file-sync/pkg/metrics"
)

var (
	client = &http.Client{}
	ts     = httptest.NewServer(metrics.GetHandler())
	ctx    = context.Background()
)

func TestMetricsInc(t *testing.T) {
	t.Parallel()

	metrics.Up.Set(1)
	metrics.QueueErrorCounter.WithLabelValues("test").Inc()
	metrics.QueueRequestCounter.WithLabelValues("test").Inc()
	metrics.QueueMaxRetryCountCounter.WithLabelValues("test").Inc()
}

func TestMetricsHandler(t *testing.T) {
	t.Parallel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, ts.URL, nil)
	if err != nil {
		t.Fatal(err)
	}

	resp, err := client.Do(req)
	if err != nil {
		t.Fatal(err)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	if m := "queue_requests_total"; !strings.Contains(string(body), m) {
		t.Fatal(fmt.Sprintf("no metric %s found", m))
	}

	if m := "queue_error_total"; !strings.Contains(string(body), m) {
		t.Fatal(fmt.Sprintf("no metric %s found", m))
	}
}
