setup_apt() {
    retry apt update
    retry apt upgrade -y

    retry apt install -y \
        apt-transport-https ca-certificates curl \
        git gnupg lsb-release iftop
}

run_or_die setup_apt
