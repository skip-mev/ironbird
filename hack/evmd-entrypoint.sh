#!/bin/sh
set -eou

echo "START_SCRIPT"
# --- Config (override via env) ---
RPC_URL="${RPC_URL:-http://localhost:8545}"
PROFILE_RATE="${PROFILE_RATE:-1}"
WAIT_TIMEOUT="${WAIT_TIMEOUT:-10}"        # seconds
PING_METHOD="${PING_METHOD:-web3_clientVersion}"

APP_PID=""

term() {
  if [ -n "${APP_PID}" ] && kill -0 "$APP_PID" 2>/dev/null; then
    kill -TERM "$APP_PID"
    wait "$APP_PID"
  fi
  exit 0
}
trap term INT TERM

jsonrpc() {
  method="$1"
  params_json="${2:-[]}"
  curl -fsS -H 'Content-Type: application/json' \
    --data "{\"jsonrpc\":\"2.0\",\"method\":\"${method}\",\"params\":${params_json},\"id\":1}" \
    "$RPC_URL"
}

wait_for_rpc() {
  start="$(date +%s)"
  while :; do
    if jsonrpc "$PING_METHOD" "[]" >/dev/null 2>&1; then
      echo "RPC ready"
      return 0
    fi
    now="$(date +%s)"
    elapsed=$((now - start))
    if [ "$elapsed" -gt "$WAIT_TIMEOUT" ]; then
      echo "Timed out (${WAIT_TIMEOUT}s) waiting for JSON-RPC at ${RPC_URL}" >&2
      return 1
    fi
    sleep 1
  done
}

# --- Start evmd ---
echo "START EVMD"
evmd "$@" &
APP_PID="$!"

# --- Wait, then enable debug profiling ---
if wait_for_rpc; then
  if ! jsonrpc "debug_setBlockProfileRate" "[${PROFILE_RATE}]" >/dev/null 2>&1; then
    echo "Warning: debug_setBlockProfileRate call failed; continuing." >&2
  else
    echo "debug_setBlockProfileRate(${PROFILE_RATE}) enabled."
  fi
else
  echo "Proceeding without enabling profiling (RPC never became ready)." >&2
fi

wait "$APP_PID"