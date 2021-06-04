package simulator

import (
	"fmt"
	"log"
	"net/http"
	"time"

	prometheusmetrics "github.com/deathowl/go-metrics-prometheus"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/rcrowley/go-metrics"
)

func ExportMetrics(config MetricsConfig) {
	prometheusClient := prometheusmetrics.NewPrometheusProvider(
		metrics.DefaultRegistry,      // go-metrics registry
		"singlestore",                // namespace
		"logistics",                  // subsystem
		prometheus.DefaultRegisterer, // prom registerer
		1*time.Second,                // flush interval
	)
	go prometheusClient.UpdatePrometheusMetrics()

	log.Printf("serving /metrics on port %d", config.Port)
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
	if err != nil {
		log.Fatalf("failed to start metrics server: %s", err)
	}
}
