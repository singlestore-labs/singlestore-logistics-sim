setup_singlestore_studio() {
    curl -s 'https://release.memsql.com/release-aug2018.gpg' | sudo apt-key add -
    echo "deb [arch=amd64] https://release.memsql.com/production/debian memsql main" >/etc/apt/sources.list.d/memsql.list

    apt update
    apt -y install singlestore-client singlestoredb-studio

    systemctl enable singlestoredb-studio
    systemctl start singlestoredb-studio
}

run_or_die setup_singlestore_studio
