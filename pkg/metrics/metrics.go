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
package metrics

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

const (
	moduleName = "filesync"
)

var (
	Up = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: moduleName,
			Name:      "up",
			Help:      "The current health status of the server (1 = UP, 0 = DOWN).",
		},
	)
	QueueSize = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: moduleName,
			Name:      "queue_size",
			Help:      "Current queue size",
		},
	)
	QueueRequestCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: moduleName,
			Name:      "queue_requests_total",
			Help:      "Number of queue request",
		},
		[]string{"type"}, // labels
	)
	QueueErrorCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: moduleName,
			Name:      "queue_error_total",
			Help:      "Number of queue errors",
		},
		[]string{"type"}, // labels
	)
	QueueMaxRetryCountCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: moduleName,
			Name:      "queue_max_retry_reach_total",
			Help:      "Number of task deleted by max_retry count",
		},
		[]string{"type"}, // labels
	)
)

func GetHandler() http.Handler {
	return promhttp.Handler()
}
