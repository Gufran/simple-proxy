#!/usr/bin/env bash

set -e

function log() {
  echo -e " >> $*"
}

function indent() {
  sed -e 's/^/    /'
}

log "Building simple proxy..."
make build | indent

config_file=$(mktemp /tmp/simple-proxy-e2e-test-XXXXXXXX.hcl)
logs_file=$(mktemp /tmp/simple-proxy-e2e-logs-XXXXXXXX)

function proxy_logs() {
  echo "    ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~ Simple Proxy Logs ~~~~~~~~~~~~~~~~~~~~~~~~~~~~~~"
  indent < "${logs_file}"
}

cat ./testing/config-1.hcl > "${config_file}"

log "Starting simple-proxy with config-1. Logs will be written to ${logs_file}."

./pkg/simple-proxy --config "${config_file}" > "${logs_file}" 2>&1 &
proxy_pid=$!

log "Simple-proxy is running with PID ${proxy_pid}"
log "Running e2e tests for config-1"

echo

go run ./testing/e2e.go --conn 20001:30001 --conn 20002:30002 --conn 20003:30003 --conn 20004:30004 |& indent
status=$?

echo

if [[ $status -ne 0 ]]; then
  log "E2E tests failed for config-1"
  proxy_logs
  exit 1
else
  log "E2E tests passed for config-1"
fi

log "Reloading configuration and running e2e tests for config-1"

cat ./testing/config-2.hcl > "${config_file}"
kill -USR1 "${proxy_pid}"

echo

go run ./testing/e2e.go --conn 40001:50001 --conn 40002:50002 --conn 40003:50003 --conn 40004:50004 |& indent
status=$?

echo

if [[ $status -ne 0 ]]; then
  log "E2E tests failed for config-2"
  proxy_logs
  exit 1
else
  log "E2E tests passed for config-2"
fi

log "Attempting graceful shutdown..."
kill -INT "${proxy_pid}"
wait "${proxy_pid}"

echo

proxy_logs

