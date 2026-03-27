#!/usr/bin/env sh

set -eu

repo_root="$(CDPATH= cd -- "$(dirname -- "$0")/.." && pwd)"
cd "$repo_root"

git config core.hooksPath .githooks

if [ -f .githooks/commit-msg ]; then
    chmod +x .githooks/commit-msg
fi

printf 'Installed git hooks: core.hooksPath=%s\n' "$(git config --get core.hooksPath)"
