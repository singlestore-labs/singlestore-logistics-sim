wait_for_s2_node() {
    local host="${1}"
    log "waiting for ${host}"
    while true; do
        if singlestore -h ${host} -proot -e "select 1" >/dev/null; then
            break
        fi
        sleep 1
    done
    log "${host} is up"
}

setup_singlestore_master() {
    local i
    local license="$(metadata s2-license)"
    local num_aggs="$(metadata s2-aggs)"
    local num_leaves="$(metadata s2-leaves)"

    memsqlctl set-license --yes --license "${license}"
    memsqlctl bootstrap-aggregator --yes --host $(hostname)
    memsqlctl enable-high-availability --yes

    memsqlctl update-config --yes --key default_partitions_per_leaf --value $(nproc) --set-global

    # setup aggs
    # start at 1 to skip the master (this node)
    for ((i = 1; i < ${num_aggs}; i++)); do
        local agg_host="s2-agg-${i}"
        wait_for_s2_node ${agg_host}
        memsqlctl add-aggregator --yes --host ${agg_host} --password root
    done

    # setup leaves
    for ((i = 0; i < ${num_leaves}; i++)); do
        local leaf_host="s2-leaf-${i}"
        wait_for_s2_node ${leaf_host}
        memsqlctl add-leaf --yes --host ${leaf_host} --password root
    done

    # initialize schema
    local schema_url="$(metadata s2-schema)"
    gsutil cp ${schema_url} /data/schema.sql
    local worldcities_url="$(metadata s2-worldcities)"
    mkdir -p /data/simplemaps
    gsutil cp ${worldcities_url} /data/simplemaps/worldcities.csv

    singlestore -proot </data/schema.sql
}

setup_singlestore_agg() {
    local node_index=$(hostname | sed 's/^.*-\([0-9]\+\)$/\1/')

    if [[ ${node_index} -eq 0 ]]; then
        run_or_die setup_singlestore_master
    fi
}

run_or_die setup_singlestore_agg
