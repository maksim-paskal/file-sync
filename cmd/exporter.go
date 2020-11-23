package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"
)

type Exporter struct {
	moduleName          string
	up                  prometheus.Gauge
	queueRequestCounter *prometheus.CounterVec
	queueErrorCounter   *prometheus.CounterVec
}

func newExporter() *Exporter {
	const moduleName = "filesync"

	return &Exporter{
		moduleName: moduleName,
		up: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: moduleName,
				Name:      "up",
				Help:      "The current health status of the server (1 = UP, 0 = DOWN).",
			},
		),
		queueRequestCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: moduleName,
				Name:      "queue_requests",
				Help:      "Number of queue request",
			},
			[]string{"type"}, // labels
		),
		queueErrorCounter: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Namespace: moduleName,
				Name:      "queue_error",
				Help:      "Number of queue errors",
			},
			[]string{"type"}, // labels
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.up.Describe(ch)
	e.queueRequestCounter.Describe(ch)
	e.queueErrorCounter.Describe(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(e.up.Desc(), prometheus.GaugeValue, 1)

	e.queueRequestCounter.Collect(ch)
	e.queueErrorCounter.Collect(ch)
}

func (e *Exporter) startServer() {
	prometheus.MustRegister(e)
	prometheus.MustRegister(version.NewCollector(e.moduleName))

	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	go func() {
		server := &http.Server{
			Addr:    *appConfig.metricsAddress,
			Handler: mux,
		}

		log.Infof("Start metrics server on %s", server.Addr)

		err := server.ListenAndServe()
		if err != nil {
			log.Panic(err)
		}
	}()
}
