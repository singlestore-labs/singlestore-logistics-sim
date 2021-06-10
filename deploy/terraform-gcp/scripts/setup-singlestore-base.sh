setup_singlestore_base() {
    cat >/etc/sysctl.d/30-singlestore.conf <<EOF
vm.max_map_count=1000000000
vm.min_free_kbytes=6710886
net.core.rmem_max=8388608
net.core.wmem_max=8388608
EOF
    sysctl --system

    echo never >/sys/kernel/mm/transparent_hugepage/enabled
    echo never >/sys/kernel/mm/transparent_hugepage/defrag
    echo 0 >/sys/kernel/mm/transparent_hugepage/khugepaged/defrag

    if [[ -d /data ]]; then
        local mem_kb=$(cat /proc/meminfo | grep MemTotal | awk '{ print int($2 * 0.15) }')
        fallocate -l ${mem_kb}KiB /data/swapfile
        chmod 600 /data/swapfile
        mkswap /data/swapfile
        swapon /data/swapfile

        mkdir -p /data/memsql /var/lib/memsql
        mount --bind /data/memsql /var/lib/memsql
    fi

    curl -s 'https://release.memsql.com/release-aug2018.gpg' | sudo apt-key add -
    echo "deb [arch=amd64] https://release.memsql.com/production/debian memsql main" >/etc/apt/sources.list.d/memsql.list

    apt update
    apt -y install singlestore-client singlestoredb-toolbox

    mkdir -p /root/.config/singlestoredb-toolbox
    echo 'user = "root"' >/root/.config/singlestoredb-toolbox/toolbox.hcl

    sdb-toolbox-config register-host --yes --localhost --cluster-hostname $(hostname)
    sdb-deploy install --yes

    memsqlctl create-node --yes --password root
}

run_or_die setup_singlestore_base
