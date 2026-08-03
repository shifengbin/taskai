#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
architecture="amd64"
generate_deb=false
deb_version="${TASKAI_DEB_VERSION:-}"
version_from_cli=false
package_workdir=""
staging_dir=""

cleanup() {
  if [[ -n "$package_workdir" && -d "$package_workdir" ]]; then
    rm -rf -- "$package_workdir"
  fi
}
trap cleanup EXIT

usage() {
  echo "用法: $0 [amd64|arm64|arm] [--deb] [--version <Debian 版本号>]" >&2
}

architecture_supplied=false
while (($# > 0)); do
  case "$1" in
    amd64|arm64|arm)
      if [[ "$architecture_supplied" == true ]]; then
        echo "Linux 架构只能指定一次。" >&2
        usage
        exit 1
      fi
      architecture="$1"
      architecture_supplied=true
      ;;
    --deb)
      generate_deb=true
      ;;
    --version)
      if (($# < 2)); then
        echo "--version 需要提供 Debian 版本号。" >&2
        usage
        exit 1
      fi
      shift
      deb_version="$1"
      version_from_cli=true
      ;;
    --*)
      echo "不支持的选项: $1" >&2
      usage
      exit 1
      ;;
    *)
      echo "不支持的 Linux 架构: $1（可选: amd64、arm64、arm）" >&2
      usage
      exit 1
      ;;
  esac
  shift
done

if [[ "$version_from_cli" == true && "$generate_deb" != true ]]; then
  echo "--version 仅可与 --deb 一起使用。" >&2
  usage
  exit 1
fi

if [[ "$(uname -s)" != "Linux" ]]; then
  echo "此脚本只能在 Linux 主机上运行。" >&2
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
if ! command -v pkg-config >/dev/null 2>&1 || ! pkg-config --exists gtk+-3.0 gio-unix-2.0; then
  echo "缺少 GTK 3 开发依赖。请安装 gtk+-3.0 开发包。" >&2
  exit 1
fi

build_tags=()
if pkg-config --exists webkit2gtk-4.1; then
  build_tags=(-tags webkit2_41)
elif ! pkg-config --exists webkit2gtk-4.0; then
  echo "缺少 WebKitGTK 开发依赖。请安装 libwebkit2gtk-4.1-dev 或 libwebkit2gtk-4.0-dev。" >&2
  exit 1
fi

if [[ "$generate_deb" == true ]]; then
  if ! command -v dpkg-deb >/dev/null 2>&1; then
    echo "未找到 dpkg-deb。请安装 Debian 打包工具。" >&2
    exit 1
  fi
  if ! command -v dpkg-shlibdeps >/dev/null 2>&1; then
    echo "未找到 dpkg-shlibdeps。请安装 dpkg-dev。" >&2
    exit 1
  fi

  if [[ -z "$deb_version" ]]; then
    git_revision="$(git -C "$project_dir" rev-parse --short HEAD 2>/dev/null || true)"
    deb_version="0.0.0+git.${git_revision:-local}"
  fi
  if ! [[ "$deb_version" =~ ^[0-9][A-Za-z0-9.+:~\-]*$ ]] || ! dpkg --validate-version "$deb_version" >/dev/null 2>&1; then
    echo "无效的 Debian 版本号: $deb_version" >&2
    exit 1
  fi
fi

cd "$project_dir"
wails build -platform "linux/$architecture" "${build_tags[@]}" -clean
echo "Linux 构建完成: $project_dir/build/bin/taskai"

if [[ "$generate_deb" != true ]]; then
  exit 0
fi

case "$architecture" in
  arm) deb_architecture="armhf" ;;
  *) deb_architecture="$architecture" ;;
esac

binary_path="$project_dir/build/bin/taskai"
icon_path="$project_dir/build/appicon.png"
package_path="$project_dir/build/bin/taskai_${deb_version}_${deb_architecture}.deb"

if [[ ! -x "$binary_path" ]]; then
  echo "未找到可执行的 Wails 构建产物: $binary_path" >&2
  exit 1
fi
if [[ ! -f "$icon_path" ]]; then
  echo "未找到应用图标: $icon_path" >&2
  exit 1
fi

package_workdir="$(mktemp -d)"
staging_dir="$package_workdir/debian/taskai"
install -d -m 755 \
  "$staging_dir/DEBIAN" \
  "$staging_dir/usr/bin" \
  "$staging_dir/usr/lib/taskai" \
  "$staging_dir/usr/share/applications" \
  "$staging_dir/usr/share/icons/hicolor/512x512/apps"
install -Dm755 "$binary_path" "$staging_dir/usr/lib/taskai/taskai"
ln -s ../lib/taskai/taskai "$staging_dir/usr/bin/taskai"
install -Dm644 "$icon_path" "$staging_dir/usr/share/icons/hicolor/512x512/apps/taskai.png"

cat > "$staging_dir/usr/share/applications/taskai.desktop" <<'EOF'
[Desktop Entry]
Type=Application
Name=TaskAI
Comment=TaskAI desktop workspace
Exec=taskai
Icon=taskai
Terminal=false
Categories=Development;Utility;
EOF
chmod 644 "$staging_dir/usr/share/applications/taskai.desktop"

cat > "$package_workdir/debian/control" <<EOF
Source: taskai
Section: utils
Priority: optional
Maintainer: TaskAI <support@taskai.invalid>
Standards-Version: 4.7.0

Package: taskai
Architecture: $deb_architecture
Description: TaskAI desktop workspace
 TaskAI desktop workspace application.
EOF

dependencies="$(cd "$package_workdir" && dpkg-shlibdeps -O -e"$staging_dir/usr/lib/taskai/taskai" | sed -n 's/^shlibs:Depends=//p')"

{
  echo "Package: taskai"
  echo "Version: $deb_version"
  echo "Architecture: $deb_architecture"
  echo "Maintainer: TaskAI <support@taskai.invalid>"
  echo "Section: utils"
  echo "Priority: optional"
  if [[ -n "$dependencies" ]]; then
    echo "Depends: $dependencies"
  fi
  echo "Description: TaskAI desktop workspace"
  echo " TaskAI desktop workspace application."
} > "$staging_dir/DEBIAN/control"
chmod 644 "$staging_dir/DEBIAN/control"

dpkg-deb --root-owner-group --build "$staging_dir" "$package_path"
echo "Linux DEB 构建完成: $package_path"
