#!/usr/bin/env bash

set -euo pipefail

_tmp_dir="$(mktemp -d)"
gorun_parent=$$
export healthcheck="https://127.0.0.1:3000/healthcheck"

cleanup() {
    echo "Exiting..."
    echo "Killing child procs of $gorun_parent != $$"
    pkill -9 -P $gorun_parent
    sleep 1
    echo "Removing ${_tmp_dir}"
    rm -rf "${_tmp_dir}"

}
trap cleanup EXIT

pushd "${_tmp_dir}" > /dev/null
    echo "Generating TLS Certificates"
    # Generate the CA Key and Certificate for signing Client Certs
    openssl req -x509 -newkey rsa:2048 -nodes \
        -keyout CAkey.pem -out CAcert.pem \
        -sha256 -days 3650 -nodes \
        -subj "/C=UK/ST=London/L=London/O=PAASDEV/CN=broker-ca.local" &> /dev/null

    # Generate the Server Key, and CSR and sign with the CA Certificate
    openssl req -new -newkey rsa:2048 -nodes \
        -keyout key.pem -out req.pem \
        -subj "/C=UK/ST=London/L=London/O=PAASDEV/CN=broker.local" &> /dev/null
    openssl x509 -req \
        -in req.pem -CA CAcert.pem -CAkey CAkey.pem \
        -extfile <(printf "subjectAltName=IP:127.0.0.1") \
        -CAcreateserial -out cert.pem -days 30 -sha256 #&> /dev/null

    # rm CAkey.pem req.pem

    cert_value="$(sed -e ':a' -e 'N' -e '$!ba' -e 's/\n/\\n/g' "cert.pem")"
    key_value="$(sed -e ':a' -e 'N' -e '$!ba' -e 's/\n/\\n/g' "key.pem")"
    ca_cert_value="$(sed -e ':a' -e 'N' -e '$!ba' -e 's/\n/\\n/g' "CAcert.pem")"
popd > /dev/null

echo "TLS Certificates Generated in ${_tmp_dir}"
echo "Will use curl command: curl --cert ${_tmp_dir}/cert.pem --key ${_tmp_dir}/key.pem --cacert ${_tmp_dir}/CAcert.pem $healthcheck"

# Interpolate the certificates with heredoc and sed (jq is not present)
cat <<-EOF > "${_tmp_dir}/config.json"
{
  "tls": {
    "certificate": "${cert_value}",
    "private_key": "${key_value}",
    "ca": "${ca_cert_value}"
  },
$(sed '1d' examples/config.json)
EOF

go run main.go --config "${_tmp_dir}/config.json" &
gorun_parent=$!

# Wait for server to start and hit healthcheck using the certs
sleep 5
set +euo pipefail # have to disable failure to read http_code
http_code=$(curl -s -o /dev/null -w "%{http_code}" --cert "${_tmp_dir}/cert.pem" --key "${_tmp_dir}/key.pem" --cacert "${_tmp_dir}/CAcert.pem" "$healthcheck")
set -euo pipefail

# Fail if code is not OK
if [ "$http_code" != "200" ]; then
    echo "Curl returned code ${http_code}, exiting."
    exit 1
fi
