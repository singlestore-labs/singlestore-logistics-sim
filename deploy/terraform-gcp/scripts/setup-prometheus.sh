setup_prometheus() {
    local simulator_targets=$(metadata simulator-targets)
    local redpanda_targets=$(metadata redpanda-targets)

    mkdir -p /data/prometheus
    chown 65534:65534 /data/prometheus

    mkdir -p /etc/prometheus
    cat >/etc/prometheus/prometheus.yml <<EOF
global:
  scrape_interval: '5s'
  evaluation_interval: '5s'

scrape_configs:
  - job_name: 'singlestore-logistics-simulator'
    static_configs:
      - targets: ${simulator_targets}
  - job_name: 'redpanda'
    static_configs:
      - targets: ${redpanda_targets}
EOF

    docker run -d \
        --name prometheus \
        --net host \
        -v /etc/prometheus/prometheus.yml:/etc/prometheus/prometheus.yml:ro \
        -v /data/prometheus:/prometheus \
        prom/prometheus:v2.27.1
}

run_or_die setup_prometheus
