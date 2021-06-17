setup_disks() {
    local mountdest=/data

    readarray -d '' local_disks < <(find /dev/disk/by-id -name "google-local-nvme-ssd-*" -print0)

    [[ ${#local_disks[@]} -eq 0 ]] && return

    local device=${local_disks[0]}

    if [[ ${#local_disks[@]} -gt 1 ]]; then
        mdadm --create /dev/md0 --level=0 --raid-devices="${#local_disks[@]}" "${local_disks[@]}"
        device=/dev/md0
    fi

    mkfs.xfs -f ${device}

    mkdir -p ${mountdest}
    mount ${device} ${mountdest}

    local mem_kb=$(cat /proc/meminfo | grep MemTotal | awk '{ print int($2 * 0.15) }')
    fallocate -l ${mem_kb}KiB ${mountdest}/swapfile
    chmod 600 ${mountdest}/swapfile
    mkswap ${mountdest}/swapfile
    swapon ${mountdest}/swapfile
}

run_or_die setup_disks
