#!/bin/bash 

set -euo pipefail

function run_command() {
    local command=$1
    local message=$2

    echo -n "${message}... "

    if ! output=$(bash -c "${command}" 2>&1); then
        echo -e "failed.\n\n"
        echo "Command Failed: ${command}"
        echo ""
        echo "Output: ${output}"
        exit 1
    fi
    echo "done."
}

command -v jq >/dev/null 2>&1 || { echo >&2 "jq is required but it's not installed.  Aborting."; exit 1; }
command -v bosh >/dev/null 2>&1 || { echo >&2 "bosh is required but it's not installed.  Aborting."; exit 1; }
command -v cut >/dev/null 2>&1 || { echo >&2 "cut is required but it's not installed.  Aborting."; exit 1; }

bosh vms --json | jq -r '.Tables[0].Rows[] | select(.instance|startswith("sqs_broker/")) | .instance' | while read -r instance; do
    run_command "bosh ssh ${instance} sudo monit stop sqs-broker" "stopping sqs-broker on ${instance}"
    run_command "bosh scp ./amd64/sqs-broker ${instance}:/tmp/sqs-broker" "copying sqs-broker binary to tmp on ${instance}"
    run_command "bosh ssh ${instance} sudo mv /tmp/sqs-broker /var/vcap/packages/sqs-broker/bin/paas-sqs-broker" "moving sqs-broker binary from tmp to packages on ${instance}"
    run_command "bosh ssh ${instance} sudo monit start sqs-broker" "starting sqs-broker on ${instance}"
done
