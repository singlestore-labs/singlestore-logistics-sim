package simulator

import (
	"fmt"
	"log"
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func ExportMetrics(config MetricsConfig) {
	log.Printf("serving /metrics on port %d", config.Port)
	http.Handle("/metrics", promhttp.Handler())
	err := http.ListenAndServe(fmt.Sprintf(":%d", config.Port), nil)
	if err != nil {
		log.Fatalf("failed to start metrics server: %s", err)
	}
}
