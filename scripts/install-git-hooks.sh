#!/usr/bin/env bash
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
REPO_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"

git -C "$REPO_ROOT" rev-parse --is-inside-work-tree >/dev/null 2>&1 || {
    echo "[X] ERROR: not a git repository: $REPO_ROOT" >&2
    exit 1
}

git -C "$REPO_ROOT" config --local core.hooksPath .githooks
echo "[OK] core.hooksPath set to .githooks"
