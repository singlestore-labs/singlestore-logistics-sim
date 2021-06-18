setup_node_exporter() {
    local version=1.1.2
    cd /tmp
    curl -L --silent https://github.com/prometheus/node_exporter/releases/download/v${version}/node_exporter-${version}.linux-amd64.tar.gz | tar -xvz
    mv /tmp/node_exporter-${version}.linux-amd64/node_exporter /usr/bin/node_exporter

    cat >/etc/systemd/system/node_exporter.service <<EOF
[Unit]
Description=Prometheus Node Exporter
After=network.target

[Service]
Restart=always
RestartSec=1
ExecStart=/usr/bin/node_exporter

[Install]
WantedBy=multi-user.target
EOF

    systemctl enable node_exporter
    systemctl daemon-reload
    systemctl start node_exporter
}

run_or_die setup_node_exporter
