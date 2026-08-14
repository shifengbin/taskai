#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
test_dir="$(mktemp -d)"
trap 'rm -rf -- "$test_dir"' EXIT

mkdir -p "$test_dir/artifacts/linux" "$test_dir/artifacts/windows" "$test_dir/artifacts/macos"
printf 'linux-package' > "$test_dir/artifacts/linux/taskai_1.2.3-rc.1_amd64.deb"
printf 'bare-windows-application' > "$test_dir/artifacts/windows/taskai.exe"
printf 'windows-package' > "$test_dir/artifacts/windows/taskai-amd64-installer.exe"
printf 'macos-package' > "$test_dir/artifacts/macos/TaskAI-1.2.3-rc.1-universal.dmg"

manifest_path="$test_dir/taskai-update.json"
bash "$project_dir/scripts/generate-update-manifest.sh" \
  1.2.3-rc.1 \
  v1.2.3-rc.1 \
  https://github.com/shifengbin/taskai/releases/tag/v1.2.3-rc.1 \
  "$test_dir/artifacts/linux" \
  "$test_dir/artifacts/windows" \
  "$test_dir/artifacts/macos" \
  "$manifest_path"

jq -e '.schemaVersion == 1' "$manifest_path" >/dev/null
jq -e '.version == "1.2.3-rc.1" and .tag == "v1.2.3-rc.1"' "$manifest_path" >/dev/null
jq -e '.assets | keys == ["darwin-universal", "linux-amd64", "windows-amd64"]' "$manifest_path" >/dev/null

for platform in linux-amd64 windows-amd64 darwin-universal; do
  jq -e --arg platform "$platform" '.assets[$platform].name | length > 0' "$manifest_path" >/dev/null
  jq -e --arg platform "$platform" '.assets[$platform].size > 0' "$manifest_path" >/dev/null
  jq -e --arg platform "$platform" '.assets[$platform].sha256 | test("^[0-9a-f]{64}$")' "$manifest_path" >/dev/null
done

jq -e '.assets["windows-amd64"].name == "taskai-amd64-installer.exe"' "$manifest_path" >/dev/null

printf 'second-installer' > "$test_dir/artifacts/windows/taskai-second-installer.exe"
if bash "$project_dir/scripts/generate-update-manifest.sh" \
  1.2.3-rc.1 \
  v1.2.3-rc.1 \
  https://github.com/shifengbin/taskai/releases/tag/v1.2.3-rc.1 \
  "$test_dir/artifacts/linux" \
  "$test_dir/artifacts/windows" \
  "$test_dir/artifacts/macos" \
  "$test_dir/ambiguous.json"; then
  echo "多个 Windows 安装包时不应生成更新清单" >&2
  exit 1
fi
rm "$test_dir/artifacts/windows/taskai-second-installer.exe"

[[ "$(bash "$project_dir/scripts/is-prerelease.sh" 1.2.3)" == "false" ]]
[[ "$(bash "$project_dir/scripts/is-prerelease.sh" 1.2.3-rc.1)" == "true" ]]
[[ "$(bash "$project_dir/scripts/is-prerelease.sh" 1.2.3-beta.2)" == "true" ]]
[[ "$(bash "$project_dir/scripts/is-prerelease.sh" 1.2.3+build-1)" == "false" ]]

if go run "$project_dir/cmd/update-manifest" \
  --version invalid_version \
  --tag vinvalid_version \
  --release-url https://github.com/shifengbin/taskai/releases/tag/vinvalid_version \
  --linux-amd64 "$test_dir/artifacts/linux/taskai_1.2.3-rc.1_amd64.deb" \
  --windows-amd64 "$test_dir/artifacts/windows/taskai-amd64-installer.exe" \
  --darwin-universal "$test_dir/artifacts/macos/TaskAI-1.2.3-rc.1-universal.dmg" \
  --output "$test_dir/invalid.json"; then
  echo "非法版本不应生成更新清单" >&2
  exit 1
fi

workflow="$project_dir/.github/workflows/build-release.yml"
grep -F 'is-prerelease' "$workflow" >/dev/null
grep -F -- '--prerelease' "$workflow" >/dev/null
grep -F 'taskai-update.json' "$workflow" >/dev/null
grep -F 'bash scripts/generate-update-manifest.sh' "$workflow" >/dev/null
grep -F 'bash scripts/is-prerelease.sh' "$workflow" >/dev/null
grep -F 'artifact-path: build/bin/*-installer.exe' "$workflow" >/dev/null
grep -F 'artifacts/taskai-windows-exe/*-installer.exe' "$workflow" >/dev/null

echo "Release 更新清单测试通过"
