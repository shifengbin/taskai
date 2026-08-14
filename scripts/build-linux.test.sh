#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d)"
trap 'rm -rf -- "$test_dir"' EXIT

test_project="$test_dir/project"
fake_bin="$test_dir/bin"
mkdir -p "$test_project/scripts" "$test_project/build" "$fake_bin"
cp "$project_dir/scripts/build-linux.sh" "$test_project/scripts/build-linux.sh"
cp "$project_dir/scripts/prepare-wails-linux-file-drop-patch.sh" "$test_project/scripts/prepare-wails-linux-file-drop-patch.sh"
mkdir -p "$test_project/patches"
cp "$project_dir/patches/wails-2.12.0-linux-file-drop.patch" "$test_project/patches/wails-2.12.0-linux-file-drop.patch"
cp "$project_dir/build/appicon.png" "$test_project/build/appicon.png"

wails_module_dir="$(go list -m -f '{{.Dir}}' github.com/wailsapp/wails/v2)"

cat > "$fake_bin/wails" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ -f "${GOWORK:-}" ]] || {
  echo "Wails 构建未收到临时 Go workspace" >&2
  exit 1
}
grep -F 'replace github.com/wailsapp/wails/v2 v2.12.0 =>' "$GOWORK" >/dev/null
[[ " $* " == *" -ldflags "* ]] || {
  echo "Wails 构建缺少 -ldflags" >&2
  exit 1
}
[[ " $* " == *"-X main.appVersion=${TASKAI_EXPECT_APP_VERSION}"* ]] || {
  echo "Wails 构建未注入预期应用版本 ${TASKAI_EXPECT_APP_VERSION}: $*" >&2
  exit 1
}
mkdir -p "$PWD/build/bin"
cp /bin/true "$PWD/build/bin/taskai"
EOF
chmod +x "$fake_bin/wails"

cat > "$fake_bin/go" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail

if [[ "$1" == "list" && "$2" == "-m" ]]; then
  if [[ "$4" == "{{.Version}}" ]]; then
    echo "v2.12.0"
  else
    echo "$TASKAI_WAILS_MODULE_DIR"
  fi
  exit 0
fi

if [[ "$1" == "work" && "$2" == "init" ]]; then
  printf 'go 1.23.0\n\nuse %s\n' "$3" > go.work
  exit 0
fi

if [[ "$1" == "work" && "$2" == "edit" && "$3" == "-replace" ]]; then
  printf 'replace github.com/wailsapp/wails/v2 v2.12.0 => %s\n' "${4#*=}" >> go.work
  exit 0
fi

exec /usr/bin/go "$@"
EOF
chmod +x "$fake_bin/go"

cat > "$fake_bin/pkg-config" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "$fake_bin/pkg-config"

run_build() {
  local expected_app_version="$1"
  shift
  PATH="$fake_bin:$PATH" \
    TASKAI_EXPECT_APP_VERSION="$expected_app_version" \
    TASKAI_WAILS_MODULE_DIR="$wails_module_dir" \
    "$test_project/scripts/build-linux.sh" "$@"
}

assert_transparent_corner() {
  local image_path="$1"
  local label="$2"
  local pixel_format
  local corner_alpha

  pixel_format="$(ffprobe -v error -show_entries stream=pix_fmt -of default=noprint_wrappers=1:nokey=1 "$image_path")"
  [[ "$pixel_format" == rgba ]] || {
    echo "$label 必须保留 alpha 通道，实际像素格式: $pixel_format" >&2
    exit 1
  }

  corner_alpha="$(ffmpeg -v error -i "$image_path" -vf 'crop=1:1:0:0' -frames:v 1 -pix_fmt rgba -f rawvideo - | od -An -tu1 | awk '{print $4}')"
  [[ "$corner_alpha" == 0 ]] || {
    echo "$label 左上角必须透明，实际 alpha: $corner_alpha" >&2
    exit 1
  }
}

assert_transparent_corner "$project_dir/build/appicon.png" "发布图标源文件"

run_build v1.2.3 --deb --version 1.2.3

package_path="$test_project/build/bin/taskai_1.2.3_amd64.deb"
[[ -f "$package_path" ]] || {
  echo "未生成带显式版本的 AMD64 DEB 包" >&2
  exit 1
}
[[ "$(dpkg-deb --field "$package_path" Package)" == "taskai" ]]
[[ "$(dpkg-deb --field "$package_path" Version)" == "1.2.3" ]]
[[ "$(dpkg-deb --field "$package_path" Architecture)" == "amd64" ]]
[[ -n "$(dpkg-deb --field "$package_path" Depends)" ]]

contents="$(dpkg-deb --contents "$package_path")"
[[ "$contents" == *"./usr/lib/taskai/taskai"* ]]
[[ "$contents" == *"./usr/bin/taskai"* ]]
[[ "$contents" == *"./usr/share/applications/taskai.desktop"* ]]
[[ "$contents" == *"./usr/share/icons/hicolor/512x512/apps/taskai.png"* ]]
[[ "$contents" == *"-rw-r--r-- root/root"*"./usr/share/applications/taskai.desktop"* ]]

desktop_file="$(dpkg-deb --fsys-tarfile "$package_path" | tar -xOf - ./usr/share/applications/taskai.desktop)"
[[ "$desktop_file" == *$'Exec=taskai\n'* ]]
[[ "$desktop_file" == *$'Icon=taskai\n'* ]]

packaged_icon="$test_dir/taskai.png"
dpkg-deb --fsys-tarfile "$package_path" | tar -xOf - ./usr/share/icons/hicolor/512x512/apps/taskai.png > "$packaged_icon"
assert_transparent_corner "$packaged_icon" "DEB 安装图标"

TASKAI_DEB_VERSION=2.3.4 run_build v2.3.4 --deb
environment_package="$test_project/build/bin/taskai_2.3.4_amd64.deb"
[[ -f "$environment_package" ]]
[[ "$(dpkg-deb --field "$environment_package" Version)" == "2.3.4" ]]

run_build v0.0.0+git.local --deb
automatic_package="$test_project/build/bin/taskai_0.0.0+git.local_amd64.deb"
[[ -f "$automatic_package" ]]
[[ "$(dpkg-deb --field "$automatic_package" Version)" == "0.0.0+git.local" ]]

if run_build vinvalid_version --deb --version 'invalid_version'; then
  echo "非法 Debian 版本号不应生成 DEB 包" >&2
  exit 1
fi
[[ ! -e "$test_project/build/bin/taskai_invalid_version_amd64.deb" ]]

echo "Linux DEB 打包集成测试通过"
