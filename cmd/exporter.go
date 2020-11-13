package main

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"
)

type Exporter struct {
	moduleName string
	up         prometheus.Gauge
	queueAdd   prometheus.Gauge
	queueErr   prometheus.Gauge
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
		queueAdd: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: moduleName,
				Name:      "queue_add",
				Help:      "Numbers of queue added",
			},
		),
		queueErr: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Namespace: moduleName,
				Name:      "queue_err",
				Help:      "Numbers of queue errors",
			},
		),
	}
}

func (e *Exporter) Describe(ch chan<- *prometheus.Desc) {
	e.up.Describe(ch)
	e.queueAdd.Describe(ch)
}

func (e *Exporter) Collect(ch chan<- prometheus.Metric) {
	ch <- prometheus.MustNewConstMetric(e.up.Desc(), prometheus.GaugeValue, 1)

	e.queueAdd.Collect(ch)
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
