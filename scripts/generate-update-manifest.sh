#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 7 ]]; then
  echo "用法: $0 <version> <tag> <release-url> <linux-dir> <windows-dir> <macos-dir> <output>" >&2
  exit 2
fi

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
version="$1"
tag="$2"
release_url="$3"
linux_dir="$4"
windows_dir="$5"
macos_dir="$6"
output="$7"

select_single_asset() {
  local directory="$1"
  local pattern="$2"
  local platform="$3"
  local matches=()

  if [[ ! -d "$directory" ]]; then
    echo "缺少 ${platform} 产物目录: $directory" >&2
    return 1
  fi
  while IFS= read -r -d '' path; do
    matches+=("$path")
  done < <(find "$directory" -maxdepth 1 -type f -name "$pattern" -print0)
  if [[ ${#matches[@]} -ne 1 ]]; then
    echo "${platform} 安装包数量必须为 1，实际为 ${#matches[@]}（匹配 $pattern）" >&2
    return 1
  fi
  printf '%s\n' "${matches[0]}"
}

linux_asset="$(select_single_asset "$linux_dir" '*.deb' 'Linux')"
windows_asset="$(select_single_asset "$windows_dir" '*-installer.exe' 'Windows')"
macos_asset="$(select_single_asset "$macos_dir" '*.dmg' 'macOS')"

go run "$project_dir/cmd/update-manifest" \
  --version "$version" \
  --tag "$tag" \
  --release-url "$release_url" \
  --linux-amd64 "$linux_asset" \
  --windows-amd64 "$windows_asset" \
  --darwin-universal "$macos_asset" \
  --output "$output"
