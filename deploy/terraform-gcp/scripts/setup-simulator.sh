setup_simulator() {
    local bin_url="$(metadata simulator-bin)"
    gsutil cp ${bin_url} /usr/bin/simulator
    chmod +x /usr/bin/simulator

    mkdir -p /etc/simulator
    cat >/etc/simulator/config.yaml <<EOF
# amount of time to simulate each tick
tick_duration: 1h

# maximum number of packages to simulate at any point (0 = unlimited)
max_packages: 100000

# number of packages to generate per tick
packages_per_tick:
  avg: 10000
  stddev: 3000

# how long packages should take to be processed
hours_at_rest:
  avg: 3
  stddev: 2

# probability a package is shipped "express"
probability_express: 0.4

# only care about shipping packages at least this far
min_shipping_distance_km: 100

# air freight is pricy - make sure a segment is far enough
min_air_freight_distance_km: 1000

# average land speed
avg_land_speed_kmph: 50

# average air speed
avg_air_speed_kmph: 750

database:
  host: s2-agg-0
  port: 3306
  username: root
  password: root
  database: logistics

topics:
  brokers:
    - rp-node-0:9092

metrics:
  port: 9000
EOF

    cat >/etc/systemd/system/simulator.service <<EOF
[Unit]
Description=SingleStore Logistics Simulator
After=network.target

[Service]
Restart=always
RestartSec=1
ExecStart=/usr/bin/simulator --config /etc/simulator/config.yaml

[Install]
WantedBy=multi-user.target
EOF

    systemctl daemon-reload
    systemctl enable simulator
    systemctl start simulator

}

run_or_die setup_simulator
