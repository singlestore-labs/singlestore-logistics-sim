setup_apt() {
    apt update
    apt upgrade -y

    apt install -y \
        apt-transport-https ca-certificates curl \
        git gnupg lsb-release iftop
}

run_or_die setup_apt