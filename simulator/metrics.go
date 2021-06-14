package simulator

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	simulationTime = promauto.NewGauge(prometheus.GaugeOpts{
		Namespace: "simulator",
		Name:      "simulation_time",
		Help:      "The latest simulation time (unix timestamp) - may not always increase as it's updated from multiple unsyncronized simulation threads",
	})
)

func ExportMetrics(config MetricsConfig) {
	log.Printf("serving /metrics on port %d", config.Port)
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
	if err != nil {
		log.Fatalf("failed to start metrics server: %s", err)
	}
}
