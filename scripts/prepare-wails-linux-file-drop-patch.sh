#!/usr/bin/env bash

set -euo pipefail

if (($# != 2)); then
  echo "用法: $0 <Wails v2.12.0 源码目录> <临时 Go workspace 目录>" >&2
  exit 1
fi

wails_dir="$1"
workspace_dir="$2"
project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
patch_file="$project_dir/patches/wails-2.12.0-linux-file-drop.patch"
source_file="$wails_dir/internal/frontend/desktop/linux/window.c"

if [[ ! -f "$source_file" ]]; then
  echo "未找到 Wails Linux 窗口源码: $source_file" >&2
  exit 1
fi
if [[ ! -f "$patch_file" ]]; then
  echo "未找到 Wails Linux 拖放补丁: $patch_file" >&2
  exit 1
fi

if patch --dry-run --forward -p1 -d "$wails_dir" < "$patch_file" >/dev/null; then
  patch --forward -p1 -d "$wails_dir" < "$patch_file" >/dev/null
elif ! patch --dry-run --reverse -p1 -d "$wails_dir" < "$patch_file" >/dev/null; then
  echo "Wails 源码与 v2.12.0 Linux 文件拖放补丁不匹配。" >&2
  exit 1
fi

(
  cd "$workspace_dir"
  go work init "$project_dir"
  go work edit -replace "github.com/wailsapp/wails/v2@v2.12.0=$wails_dir"
)
