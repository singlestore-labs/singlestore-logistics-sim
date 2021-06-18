tune_machine() {
    mkdir -p /data/redpanda /var/lib/redpanda
    mount --bind /data/redpanda /var/lib/redpanda

    curl -1sLf 'https://packages.vectorized.io/nzc4ZYQK3WRGd9sy/redpanda/cfg/setup/bash.deb.sh' | bash
    apt install -y redpanda

    rpk redpanda mode production
    rpk redpanda tune all
}

run_or_die tune_machine
