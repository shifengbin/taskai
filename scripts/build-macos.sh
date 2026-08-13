#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
architecture="universal"
generate_dmg=false
dmg_version="${TASKAI_VERSION:-}"
version_from_cli=false
staging_dir=""

cleanup() {
  if [[ -n "$staging_dir" && -d "$staging_dir" ]]; then
    rm -rf -- "$staging_dir"
  fi
}
trap cleanup EXIT

usage() {
  echo "用法: $0 [universal|amd64|arm64] [--dmg] [--version <版本>]" >&2
}

architecture_supplied=false
while (($# > 0)); do
  case "$1" in
    amd64|arm64|universal)
      if [[ "$architecture_supplied" == true ]]; then
        echo "macOS 架构只能指定一次。" >&2
        usage
        exit 1
      fi
      architecture="$1"
      architecture_supplied=true
      ;;
    --dmg)
      generate_dmg=true
      ;;
    --version)
      if (($# < 2)); then
        echo "--version 需要提供版本号。" >&2
        usage
        exit 1
      fi
      shift
      dmg_version="$1"
      version_from_cli=true
      ;;
    --*)
      echo "不支持的选项: $1" >&2
      usage
      exit 1
      ;;
    *)
      echo "不支持的 macOS 架构: $1（可选: universal、amd64、arm64）" >&2
      usage
      exit 1
      ;;
  esac
  shift
done

if [[ "$version_from_cli" == true && "$generate_dmg" != true ]]; then
  echo "--version 仅可与 --dmg 一起使用。" >&2
  usage
  exit 1
fi

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "此脚本只能在 macOS 主机上运行。" >&2
  exit 1
fi
if ! command -v wails >/dev/null 2>&1; then
  echo "未找到 Wails CLI。请先安装 Wails v2。" >&2
  exit 1
fi
if ! command -v go >/dev/null 2>&1; then
  echo "未找到 Go。请安装项目所需的 Go 工具链。" >&2
  exit 1
fi
if ! command -v xcodebuild >/dev/null 2>&1; then
  echo "未找到 Xcode 命令行工具。请先执行 xcode-select --install。" >&2
  exit 1
fi

if [[ "$generate_dmg" == true ]]; then
  if ! command -v hdiutil >/dev/null 2>&1; then
    echo "未找到 hdiutil。DMG 打包需要 macOS 自带的 hdiutil。" >&2
    exit 1
  fi

  if [[ -z "$dmg_version" ]]; then
    git_revision="$(git -C "$project_dir" rev-parse --short HEAD 2>/dev/null || true)"
    dmg_version="0.0.0+git.${git_revision:-local}"
  fi
  if ! [[ "$dmg_version" =~ ^[0-9][A-Za-z0-9.+:~-]*$ ]]; then
    echo "无效的版本号: $dmg_version" >&2
    exit 1
  fi
fi

cd "$project_dir"
wails build -platform "darwin/$architecture" -clean
app_path="$project_dir/build/bin/taskai.app"
echo "macOS 构建完成: $app_path"

if [[ "$generate_dmg" != true ]]; then
  exit 0
fi

if [[ ! -d "$app_path" ]]; then
  echo "未找到应用包: $app_path" >&2
  exit 1
fi

package_path="$project_dir/build/bin/TaskAI-${dmg_version}-${architecture}.dmg"

staging_dir="$(mktemp -d)"
cp -R "$app_path" "$staging_dir/"
ln -s /Applications "$staging_dir/Applications"

rm -f "$package_path"
hdiutil create \
  -volname "TaskAI" \
  -srcfolder "$staging_dir" \
  -fs HFS+ \
  -format UDZO \
  "$package_path"

echo "macOS DMG 构建完成: $package_path"
