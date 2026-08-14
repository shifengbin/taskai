# Windows Release NSIS Fix Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** 让 Windows GitHub Actions 构建稳定生成 NSIS 安装包，并发布包含三端安装包和更新清单的 `v0.0.0-rc7`。

**Architecture:** 在 Windows 构建任务进入 Wails 前安装并验证 `makensis`，从依赖入口消除 Wails 静默跳过安装包的问题。保留现有严格的 `*-installer.exe` 资产选择，并用发布脚本测试固定工作流依赖。

**Tech Stack:** GitHub Actions YAML、PowerShell、Chocolatey、Wails v2.12.0、Bash 测试脚本、GitHub REST API。

---

### Task 1: 增加 NSIS 依赖回归测试

**Files:**
- Modify: `scripts/build-release.test.sh:68`
- Test: `scripts/build-release.test.sh`

**Step 1: Write the failing test**

在工作流断言末尾加入：

```bash
grep -F 'choco install nsis --no-progress --yes' "$workflow" >/dev/null
grep -F 'Get-Command makensis -ErrorAction Stop' "$workflow" >/dev/null
```

**Step 2: Run test to verify it fails**

Run: `bash scripts/build-release.test.sh`

Expected: FAIL，因为当前工作流没有安装或验证 NSIS。

### Task 2: 安装并验证 Windows NSIS

**Files:**
- Modify: `.github/workflows/build-release.yml:88`
- Test: `scripts/build-release.test.sh`

**Step 1: Write minimal implementation**

在 Windows 构建脚本测试之前加入：

```yaml
      - name: Install NSIS
        if: matrix.os == 'windows-latest'
        shell: pwsh
        run: |
          choco install nsis --no-progress --yes
          Get-Command makensis -ErrorAction Stop
```

**Step 2: Run test to verify it passes**

Run: `bash scripts/build-release.test.sh`

Expected: PASS，输出 `Release 更新清单测试通过`。

**Step 3: Commit**

```bash
git add .github/workflows/build-release.yml scripts/build-release.test.sh
git commit -m "fix: 安装 Windows 发布所需 NSIS"
```

### Task 3: 执行发布前完整验证

**Files:**
- Verify: `.github/workflows/build-release.yml`
- Verify: `scripts/build-release.test.sh`

**Step 1: Run backend verification**

Run: `go test -race ./... && go test -tags updater_integration ./...`

Expected: PASS。

**Step 2: Run frontend verification**

Run: `cd frontend && npm test && npm run build`

Expected: 17 个测试文件、349 项测试全部通过，生产构建完成。

**Step 3: Run packaging and specification verification**

Run: `bash scripts/prepare-wails-linux-file-drop-patch.test.sh && bash scripts/build-release.test.sh && bash scripts/build-linux.test.sh`

Run: `openspec validate application-auto-update --type spec --strict && openspec validate platform-release-packaging --type spec --strict && git diff --check`

Expected: PASS。

### Task 4: 合并修复并发布 rc7

**Files:**
- Merge: branch `fix/windows-release-nsis` into `main`
- Tag: `v0.0.0-rc7`

**Step 1: Synchronize and merge**

将最新 `main` 合并到修复分支并复验发布脚本测试，再把修复分支合并回 `main`。

**Step 2: Push main and tag**

Run: `git push origin main`

Run: `git tag v0.0.0-rc7 && git push origin v0.0.0-rc7`

Expected: GitHub Actions `build-release` 工作流由 `v0.0.0-rc7` 触发。

**Step 3: Verify release assets**

持续检查工作流直到完成，确认 Release 为 Prerelease，且唯一包含当前版本的 DEB、`*-installer.exe`、DMG 和 `taskai-update.json`。

**Step 4: Clean up**

删除已合并的修复 worktree 与分支，确认用户的 `openspec/changes/add-gitlab-project-batch-import/` 未被提交或修改。
