#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
if ! command -v pwsh >/dev/null 2>&1; then
  echo "跳过：Windows 构建脚本测试需要 PowerShell。"
  exit 0
fi

pwsh -NoProfile -File "$project_dir/scripts/build-windows.test.ps1"
