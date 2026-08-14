#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 1 ]]; then
  echo "用法: $0 <semantic-version>" >&2
  exit 2
fi

version_without_metadata="${1%%+*}"
if [[ "$version_without_metadata" == *-* ]]; then
  printf 'true\n'
else
  printf 'false\n'
fi
