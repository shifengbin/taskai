#!/usr/bin/env bash

set -euo pipefail

if [[ "$(uname -s)" != "Darwin" ]]; then
  echo "跳过：DMG 打包测试仅在 macOS 上运行。"
  exit 0
fi

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d)"
trap 'rm -rf -- "$test_dir"' EXIT

test_project="$test_dir/project"
fake_bin="$test_dir/bin"
mkdir -p "$test_project/scripts" "$test_project/build" "$fake_bin"
cp "$project_dir/scripts/build-macos.sh" "$test_project/scripts/build-macos.sh"

# 伪装的 wails：被调用时在 build/bin 下生成一个最小可用的 .app 包。
cat > "$fake_bin/wails" <<'EOF'
#!/usr/bin/env bash
set -euo pipefail
[[ " $* " == *" -ldflags "* ]] || {
  echo "Wails 构建缺少 -ldflags" >&2
  exit 1
}
[[ " $* " == *"-X main.appVersion=${TASKAI_EXPECT_APP_VERSION}"* ]] || {
  echo "Wails 构建未注入预期应用版本 ${TASKAI_EXPECT_APP_VERSION}: $*" >&2
  exit 1
}
app_dir="$PWD/build/bin/taskai.app"
mkdir -p "$app_dir/Contents/MacOS"
cat > "$app_dir/Contents/Info.plist" <<'PLIST'
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0"><dict><key>CFBundleExecutable</key><string>taskai</string></dict></plist>
PLIST
printf '#!/usr/bin/env bash\necho taskai\n' > "$app_dir/Contents/MacOS/taskai"
chmod +x "$app_dir/Contents/MacOS/taskai"
EOF
chmod +x "$fake_bin/wails"

# 伪装的 go 与 xcodebuild：脚本只检查它们存在，不需要真正执行。
cat > "$fake_bin/go" <<'EOF'
#!/usr/bin/env bash
exec /usr/bin/go "$@"
EOF
chmod +x "$fake_bin/go"

cat > "$fake_bin/xcodebuild" <<'EOF'
#!/usr/bin/env bash
exit 0
EOF
chmod +x "$fake_bin/xcodebuild"

run_build() {
  local expected_app_version="$1"
  shift
  PATH="$fake_bin:$PATH" TASKAI_EXPECT_APP_VERSION="$expected_app_version" "$test_project/scripts/build-macos.sh" "$@"
}

# 1. 显式版本 → 带版本号的 DMG，且镜像内含 .app 与 /Applications 软链接。
run_build v1.2.3 --dmg --version 1.2.3
package_path="$test_project/build/bin/TaskAI-1.2.3-universal.dmg"
[[ -f "$package_path" ]] || { echo "未生成带显式版本的 DMG 包" >&2; exit 1; }

mount_point="$test_dir/mount"
mkdir -p "$mount_point"
hdiutil attach "$package_path" -mountpoint "$mount_point" -nobrowse -readonly >/dev/null
[[ -d "$mount_point/taskai.app" ]] || { echo "DMG 内缺少 taskai.app" >&2; exit 1; }
[[ -L "$mount_point/Applications" ]] || { echo "DMG 内缺少 /Applications 软链接" >&2; exit 1; }
hdiutil detach "$mount_point" >/dev/null

# 2. TASKAI_VERSION 环境变量。
TASKAI_VERSION=2.3.4 run_build v2.3.4 --dmg
[[ -f "$test_project/build/bin/TaskAI-2.3.4-universal.dmg" ]] || { echo "未按 TASKAI_VERSION 生成 DMG" >&2; exit 1; }

# 3. 默认版本（非 git 仓库 → 0.0.0+git.local）。
run_build v0.0.0+git.local --dmg
[[ -f "$test_project/build/bin/TaskAI-0.0.0+git.local-universal.dmg" ]] || { echo "未生成默认版本 DMG" >&2; exit 1; }

# 4. 非法版本号 → 失败，且不产出 DMG。
if run_build vinvalid_version --dmg --version 'invalid_version'; then
  echo "非法版本号不应生成 DMG" >&2
  exit 1
fi
[[ ! -e "$test_project/build/bin/TaskAI-invalid_version-universal.dmg" ]]

# 5. 不带 --dmg → 仅产出 .app，不产出 DMG。
rm -rf "$test_project/build/bin"
run_build v0.0.0+git.local
[[ -d "$test_project/build/bin/taskai.app" ]] || { echo "未启用 --dmg 时应仍产出 .app" >&2; exit 1; }
shopt -s nullglob
dmgs=( "$test_project/build/bin/"*.dmg )
shopt -u nullglob
[[ ${#dmgs[@]} -eq 0 ]] || { echo "未启用 --dmg 时不应产出 DMG" >&2; exit 1; }

echo "macOS DMG 打包集成测试通过"
