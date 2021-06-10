#!/usr/bin/env bash
set -euo pipefail

METADATA_BASE="http://metadata.google.internal/computeMetadata/v1/instance/attributes"

log() {
    local msg="${*}"
    logger -p syslog.info -t "startup-script" -- "${msg}"
}

fail() {
    local msg="${*}"
    logger -p syslog.error -t "startup-script" -- "${msg}"
    exit 1
}

run_or_die() {
    local argv=("${@}")
    log "run_or_die(${argv[*]})"
    "${argv[@]}"
}

metadata() {
    local key="${1}"
    curl --silent --connect-timeout 5 --fail \
        -H "Metadata-Flavor: Google" \
        "${METADATA_BASE}/${key}"
}
