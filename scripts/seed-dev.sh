#!/usr/bin/env bash

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SOURCE_DIR="${DEV_SAMPLE_DIR:-"$ROOT_DIR/example/dev"}"
TARGET_DIR="${1:?usage: seed-dev.sh <config-dir>}"

mkdir -p "$TARGET_DIR"

if [[ -n "$(find "$TARGET_DIR" -mindepth 1 -maxdepth 1 -print -quit)" ]]; then
    printf 'dev sample: keep existing config directory: %s\n' "$TARGET_DIR"
    exit 0
fi

if [[ ! -d "$SOURCE_DIR" ]]; then
    printf 'dev sample: source directory does not exist: %s\n' "$SOURCE_DIR" >&2
    exit 1
fi

cp -a "$SOURCE_DIR"/. "$TARGET_DIR"/
printf 'dev sample: initialized config directory: %s\n' "$TARGET_DIR"
