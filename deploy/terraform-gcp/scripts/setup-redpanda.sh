setup_redpanda() {
    local node_index=$(hostname | sed 's/^.*-\([0-9]\+\)$/\1/')
    local partitions_per_topic="$(metadata partitions-per-topic)"

    # probably installed via tune-machine.sh
    if ! dpkg -s redpanda; then
        mkdir -p /data/redpanda /var/lib/redpanda
        mount --bind /data/redpanda /var/lib/redpanda

        curl -1sLf 'https://packages.vectorized.io/nzc4ZYQK3WRGd9sy/redpanda/cfg/setup/bash.deb.sh' | bash
        apt install -y redpanda

        rpk redpanda mode production
        rpk redpanda tune all
    fi

    rpk config set cluster_id redpanda
    rpk config set organization singlestore
    rpk config set redpanda.default_topic_partitions ${partitions_per_topic}
    rpk config set redpanda.enable_idempotence true

    if [[ ${node_index} -eq 0 ]]; then
        rpk config bootstrap --id 0 --self $(hostname -i)
    else
        rpk config bootstrap --id ${node_index} --self $(hostname -i) --ips $(dig +search +short rp-node-0)
    fi

    systemctl start redpanda-tuner redpanda

    # create the topics
    rpk topic create --replicas 1 --partitions ${partitions_per_topic} packages
    rpk topic create --replicas 1 --partitions ${partitions_per_topic} transitions
}

run_or_die setup_redpanda
