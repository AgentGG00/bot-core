#!/usr/bin/env bash
set -euo pipefail
SOURCE_DIR="$1"
mkdir -p "$SOURCE_DIR/state"
touch "$SOURCE_DIR/state/lockdown.flag"
