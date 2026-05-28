#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ENV_FILE="${ENV_FILE:-"$ROOT_DIR/.env"}"
BINARY="$ROOT_DIR/build/clash-composer"

fail() {
    printf 'deploy: %s\n' "$*" >&2
    exit 1
}

require_command() {
    command -v "$1" >/dev/null 2>&1 || fail "missing required command: $1"
}

require_env() {
    local name="$1"
    if [[ -z "${!name:-}" ]]; then
        fail "missing required environment value: $name"
    fi
}

if [[ ! -f "$ENV_FILE" ]]; then
    fail "missing env file: $ENV_FILE"
fi

set -a
# shellcheck disable=SC1090
. "$ENV_FILE"
set +a

DEPLOY_GOOS="${DEPLOY_GOOS:-linux}"
DEPLOY_GOARCH="${DEPLOY_GOARCH:-arm64}"
DEPLOY_CGO_ENABLED="${DEPLOY_CGO_ENABLED:-0}"
DEPLOY_REMOTE_TMP="${DEPLOY_REMOTE_TMP:-/tmp/clash-composer.deploy}"

require_env DEPLOY_HOST
require_env DEPLOY_USER
require_env DEPLOY_SSH_PASSWORD
require_env DEPLOY_SUDO_PASSWORD
require_env DEPLOY_REMOTE_BIN
require_env DEPLOY_SERVICE

require_command make
require_command sshpass
require_command scp
require_command ssh

target="${DEPLOY_USER}@${DEPLOY_HOST}"
ssh_opts=(-o StrictHostKeyChecking=accept-new)

printf 'Building %s/%s binary...\n' "$DEPLOY_GOOS" "$DEPLOY_GOARCH"
(
    cd "$ROOT_DIR"
    GOOS="$DEPLOY_GOOS" GOARCH="$DEPLOY_GOARCH" CGO_ENABLED="$DEPLOY_CGO_ENABLED" make build
)

if [[ ! -x "$BINARY" ]]; then
    fail "build output is not executable: $BINARY"
fi

printf 'Uploading to %s:%s...\n' "$target" "$DEPLOY_REMOTE_TMP"
export SSHPASS="$DEPLOY_SSH_PASSWORD"
sshpass -e scp "${ssh_opts[@]}" "$BINARY" "$target:$DEPLOY_REMOTE_TMP"

remote_tmp_q="$(printf '%q' "$DEPLOY_REMOTE_TMP")"
remote_bin_q="$(printf '%q' "$DEPLOY_REMOTE_BIN")"
service_q="$(printf '%q' "$DEPLOY_SERVICE")"
remote_cmd="set -euo pipefail; sudo -S -p '' install -m 0755 $remote_tmp_q $remote_bin_q; systemctl --user restart $service_q; systemctl --user is-active $service_q; $remote_bin_q --help >/dev/null"
remote_cmd_q="$(printf '%q' "$remote_cmd")"

printf 'Installing and restarting %s on %s...\n' "$DEPLOY_SERVICE" "$target"
printf '%s\n' "$DEPLOY_SUDO_PASSWORD" | sshpass -e ssh -tt "${ssh_opts[@]}" "$target" "bash -lc $remote_cmd_q"

printf 'Deploy complete: %s on %s\n' "$DEPLOY_REMOTE_BIN" "$target"
