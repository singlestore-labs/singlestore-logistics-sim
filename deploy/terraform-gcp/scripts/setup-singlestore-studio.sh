setup_singlestore_studio() {
    curl -s 'https://release.memsql.com/release-aug2018.gpg' | sudo apt-key add -
    echo "deb [arch=amd64] https://release.memsql.com/production/debian memsql main" >/etc/apt/sources.list.d/memsql.list

    apt update
    apt -y install singlestore-client singlestoredb-studio

    cat >/var/lib/singlestoredb-studio/studio.hcl <<EOF
cluster "logistics" {
    name              = "logistics"
    description       = "logistics simulation"
    hostname          = "s2-agg-0"
    port              = 3306
    profile           = "PRODUCTION"
    websocket         = false
    websocketSSL      = false
    kerberosAutologin = false
}
EOF

    systemctl enable singlestoredb-studio
    systemctl start singlestoredb-studio
}

run_or_die setup_singlestore_studio
