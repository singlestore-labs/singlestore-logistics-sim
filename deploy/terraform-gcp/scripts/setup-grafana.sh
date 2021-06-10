setup_grafana() {
    local dashboards=$(metadata dashboards)

    mkdir -p /etc/grafana/provisioning/{datasources,dashboards,plugins,notifiers}
    mkdir -p /etc/grafana/dashboards/

    cat >/etc/grafana/provisioning/datasources/all.yaml <<EOF
apiVersion: 1

datasources:
  - name: prometheus
    access: proxy
    editable: false
    isDefault: true
    type: prometheus
    url: 'http://localhost:9090'
    version: 1
    jsonData:
      timeInterval: 5s
  - name: singlestore
    editable: false
    type: mysql
    url: s2-agg-0:3306
    database: logistics
    user: root
    version: 1
    jsonData:
      timeInterval: 5s
    secureJsonData:
      password: root
EOF

    cat >/etc/grafana/provisioning/dashboards/all.yaml <<EOF
apiVersion: 1

providers:
  - name: singlestore-logistics-sim
    orgId: 1
    disableDeletion: false
    updateIntervalSeconds: 10
    # to update a dashboard use "Save JSON to file" and wait for this provider to sync
    allowUiUpdates: false
    options:
      path: /etc/grafana/dashboards
      foldersFromFilesStructure: true
EOF

    for dashboard in ${dashboards}; do
        gsutil cp ${dashboard} /etc/grafana/dashboards/
    done

    docker run -d \
        --name grafana \
        --net host \
        -e "GF_INSTALL_PLUGINS=grafana-worldmap-panel,https://github.com/WilliamVenner/grafana-timepicker-buttons/releases/download/v4.1.1/williamvenner-timepickerbuttons-panel-4.1.1.zip;grafana-timepicker-buttons" \
        -v /etc/grafana/provisioning:/etc/grafana/provisioning:ro \
        -v /etc/grafana/provisioning:/etc/grafana/provisioning:ro \
        -v /etc/grafana/dashboards:/etc/grafana/dashboards:ro \
        grafana/grafana:7.5.7
}

run_or_die setup_grafana