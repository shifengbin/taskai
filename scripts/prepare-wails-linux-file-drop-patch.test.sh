#!/usr/bin/env bash

set -euo pipefail

project_dir="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
module_dir="$(go list -m -f '{{.Dir}}' github.com/wailsapp/wails/v2)"
test_dir="$(mktemp -d)"
trap 'rm -rf -- "$test_dir"' EXIT

patched_module="$test_dir/wails"
cp -a "$module_dir" "$patched_module"
chmod -R u+w "$patched_module"

mkdir -p "$test_dir/workspace"
"$project_dir/scripts/prepare-wails-linux-file-drop-patch.sh" "$patched_module" "$test_dir/workspace"

patched_source="$patched_module/internal/frontend/desktop/linux/window.c"
drag_drop_block="$(sed -n '/static gboolean onDragDrop/,/^}/p' "$patched_source")"

grep -F 'char *droppedFiles = NULL;' "$patched_source" >/dev/null
[[ "$drag_drop_block" == *'if(droppedFiles == NULL)'* ]]
[[ "$drag_drop_block" == *'processMessage(res);'* ]]
[[ "$drag_drop_block" == *$'processMessage(res);\n    free(res);'* ]]
[[ "$drag_drop_block" == *'gtk_drag_finish(context, TRUE, FALSE, time);'* ]]
[[ "$drag_drop_block" == *'return TRUE;'* ]]
if [[ "$drag_drop_block" == *'gtk_drag_get_data('* ]]; then
  echo "Wails Linux 拖放补丁不应替换原有的路径收集流程" >&2
  exit 1
fi

resolved_module_dir="$(GOWORK="$test_dir/workspace/go.work" go list -m -f '{{.Dir}}' github.com/wailsapp/wails/v2)"
[[ "$resolved_module_dir" == "$patched_module" ]]

echo "Wails Linux 文件拖放补丁测试通过"
