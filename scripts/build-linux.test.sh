#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d)"
trap 'rm -rf -- "$test_dir"' EXIT

test_project="$test_dir/project"
fake_bin="$test_dir/bin"
mkdir -p "$test_project/scripts" "$test_project/build" "$fake_bin"
cp "$project_dir/scripts/build-linux.sh" "$test_project/scripts/build-linux.sh"
cp "$project_dir/build/appicon.png" "$test_project/build/appicon.png"

cat > "$fake_bin/wails" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
mkdir -p "$PWD/build/bin"
cp /bin/true "$PWD/build/bin/taskai"
EOF
chmod +x "$fake_bin/wails"

cat > "$fake_bin/pkg-config" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "$fake_bin/pkg-config"

run_build() {
  PATH="$fake_bin:$PATH" "$test_project/scripts/build-linux.sh" "$@"
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

run_build --deb --version 1.2.3

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

TASKAI_DEB_VERSION=2.3.4 run_build --deb
environment_package="$test_project/build/bin/taskai_2.3.4_amd64.deb"
[[ -f "$environment_package" ]]
[[ "$(dpkg-deb --field "$environment_package" Version)" == "2.3.4" ]]

run_build --deb
automatic_package="$test_project/build/bin/taskai_0.0.0+git.local_amd64.deb"
[[ -f "$automatic_package" ]]
[[ "$(dpkg-deb --field "$automatic_package" Version)" == "0.0.0+git.local" ]]

if run_build --deb --version 'invalid_version'; then
  echo "非法 Debian 版本号不应生成 DEB 包" >&2
  exit 1
fi
[[ ! -e "$test_project/build/bin/taskai_invalid_version_amd64.deb" ]]

echo "Linux DEB 打包集成测试通过"
