setup_simulator() {
    local bin_url="$(metadata simulator-bin)"
    gsutil cp ${bin_url} /usr/bin/simulator
}

run_or_die setup_simulator
