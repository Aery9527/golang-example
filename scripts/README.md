# `scripts/` 目錄說明

## 快速導覽

- [概覽](#概覽)
- [檔案總覽](#檔案總覽)
- [測試與驗證腳本](#測試與驗證腳本)
- [Git 與 Agent 輔助腳本](#git-與-agent-輔助腳本)
- [Release 腳本](#release-腳本)
- [`tests/` 子目錄](#tests-子目錄)
- [常見用法](#常見用法)

## 概覽

[`scripts/`](.) 收納這個 repo 的開發輔助腳本，主要分成四類：

1. 測試入口與共用 runner
2. Git hooks 與 agent skills 輔助工具
3. release notes 產生器
4. 腳本本身的 regression tests

大多數跨平台能力都同時提供 Bash 與 PowerShell 版本；其中 [`go-test.sh`](go-test.sh) / [`go-test.ps1`](go-test.ps1) 是測試 runner 本體，其他 `ci-test` / `dev-test` 多半只是包裝入口。

[返回開頭](#快速導覽)

## 檔案總覽

| 路徑 | 類型 | 主要用途 | 備註 |
| --- | --- | --- | --- |
| [ci-test.sh](ci-test.sh) | Bash | 執行 CI 範圍的快速測試 | 包裝 [go-test.sh](go-test.sh) 的 `ci` mode |
| [ci-test.ps1](ci-test.ps1) | PowerShell | 執行 CI 範圍的快速測試 | 包裝 [go-test.ps1](go-test.ps1) 的 `ci` mode |
| [dev-test.sh](dev-test.sh) | Bash | 執行開發期較完整的 scoped 測試 | 包裝 [go-test.sh](go-test.sh) 的 `dev` mode |
| [dev-test.ps1](dev-test.ps1) | PowerShell | 執行開發期較完整的 scoped 測試 | 包裝 [go-test.ps1](go-test.ps1) 的 `dev` mode |
| [go-test.sh](go-test.sh) | Bash | 真正負責組裝 `go test`、驗證參數、落 test artifacts | 一般由 `ci-test` / `dev-test` 呼叫 |
| [go-test.ps1](go-test.ps1) | PowerShell | PowerShell 版共用 test runner | 功能與 [go-test.sh](go-test.sh) 對齊 |
| [install-git-hooks.sh](install-git-hooks.sh) | Bash | 將 repo-local Git hooks 安裝到目前 repo | 設定 `core.hooksPath=.githooks` |
| [install-git-hooks.ps1](install-git-hooks.ps1) | PowerShell | 安裝 repo-local Git hooks | 設定 `core.hooksPath=.githooks` |
| [link-agent-skills.sh](link-agent-skills.sh) | Bash | 互動式建立 `.agents/skills` 與 [../.claude/skills](../.claude/skills) 的 symlink | 適合 Git Bash / Unix-like 環境 |
| [link-agent-skills.ps1](link-agent-skills.ps1) | PowerShell | 互動式建立 `.agents/skills` 與 [../.claude/skills](../.claude/skills) 的 junction | 適合 Windows |
| [link-agent-skills.md](link-agent-skills.md) | Markdown | 補充說明 `link-agent-skills` 兩支腳本的操作方式 | 細節文件，不是可執行腳本 |
| [release-notes.py](release-notes.py) | Python | 從 Git history 收集 Conventional Commits，產生 release note 原始資料 | 預設輸出 JSON，也支援 Markdown |
| [tests/](tests) | 目錄 | 放腳本層級的 regression tests | 目前包含 `release-notes` 與 init template 測試 |

[返回開頭](#快速導覽)

## 測試與驗證腳本

### `ci-test`：快速驗證入口

[`ci-test.sh`](ci-test.sh) 與 [`ci-test.ps1`](ci-test.ps1) 都只是薄包裝：

- 固定用 `ci` mode 呼叫共用 runner
- 測試目標固定為 `./internal/...` 與 `./pkg/...`
- 會附加 `-short`
- 不產生 coverage artifacts

適合：

- pre-push gate
- 改小範圍程式後想先快速確認沒有明顯 regression

### `dev-test`：開發期驗證入口

[`dev-test.sh`](dev-test.sh) 與 [`dev-test.ps1`](dev-test.ps1) 也都是共用 runner 的包裝入口，但使用 `dev` mode：

- 不附加 `-short`
- 會產生 coverage profile 與 coverage summary
- 同樣只測 `./internal/...` 與 `./pkg/...`

適合：

- 開發中需要看 coverage
- 想比 `ci-test` 更完整地驗證目前修改

### `go-test`：共用 runner 本體

[`go-test.sh`](go-test.sh) 與 [`go-test.ps1`](go-test.ps1) 負責真正的 runner 邏輯：

- 清空並重建 `test-output/<mode>-test/`
- 固定測試 scope 為 `./internal/...` 與 `./pkg/...`
- 驗證 extra args，避免不合法用法（例如 `-args`、自訂 `-coverprofile`、超出 scope 的 `-coverpkg`）
- 將實際執行命令寫入 `command.txt`
- 將 exit code 寫入 `exit-code.txt`
- 根據 `-json` 決定 standard output 寫入 `stdout.log` 或 `stdout.jsonl`
- 永遠保留 `stderr.log`
- `dev` mode 會額外產生 `coverage.out` 與 `coverage-summary.txt`

如果只是日常開發，通常直接用 [`ci-test.sh`](ci-test.sh) / [`ci-test.ps1`](ci-test.ps1) 或 [`dev-test.sh`](dev-test.sh) / [`dev-test.ps1`](dev-test.ps1) 就夠了，不需要直接手敲 [`go-test.sh`](go-test.sh) / [`go-test.ps1`](go-test.ps1)。

[返回開頭](#快速導覽)

## Git 與 Agent 輔助腳本

### 安裝 repo-local Git hooks

[`install-git-hooks.sh`](install-git-hooks.sh) 與 [`install-git-hooks.ps1`](install-git-hooks.ps1) 的責任很單純：

- 確認目前在 Git repository 內
- 設定 local Git config：`core.hooksPath=.githooks`

這樣之後 `git push` 時，Git 就會使用 [../.githooks/pre-push](../.githooks/pre-push)。

### 連結 agent skills

[`link-agent-skills.sh`](link-agent-skills.sh) 與 [`link-agent-skills.ps1`](link-agent-skills.ps1) 用來把 [../.claude/skills](../.claude/skills) 暴露到 `.agents/skills`：

- Bash 版本建立 **symlink**
- PowerShell 版本建立 **junction**
- 兩者都提供互動式選單
- 兩者都會同步維護 `.gitignore`
- 支援整個目錄單一連結，或逐個 skill 建立連結
- 支援解除連結與清理失效項目

若想看更完整的操作說明與模式差異，直接看 [`link-agent-skills.md`](link-agent-skills.md)。

[返回開頭](#快速導覽)

## Release 腳本

[`release-notes.py`](release-notes.py) 是 release workflow 的資料收集器。它會讀 Git history，挑出 Conventional Commits，並做以下事情：

- 自動解析上一個 tag，或接受手動指定的 revision range
- 忽略 `Merge ...`、`merge:`、`release:` 這類 release noise
- 依 `feat`、`fix`、`perf`、`docs` 等群組分類 commit
- 擷取 `BREAKING CHANGE:` footer
- 預設將結果寫到 `.tmp/release/raw-commits.json`
- 使用 `--format=markdown` 時，改為直接輸出可讀的 Markdown release notes

這支腳本主要提供給 release workflow 與 agent skill 使用，但也可以獨立手動執行。

[返回開頭](#快速導覽)

## `tests/` 子目錄

[`tests/`](tests) 目前放兩支 Python `unittest`：

| 檔案 | 主要用途 | 備註 |
| --- | --- | --- |
| [tests/test_release_notes.py](tests/test_release_notes.py) | 驗證 [`release-notes.py`](release-notes.py) 在暫存 Git repo 中的分類、tag range、breaking change 與 Markdown 輸出行為 | 這支測試會保留在 init 後的新 repo |
| [tests/test_init_templates.py](tests/test_init_templates.py) | 驗證 root [../init.sh](../init.sh) / [../init.ps1](../init.ps1) 的 template 行為，例如 generated `main.go` 流程、PowerShell 非 git 目錄執行、`scripts/tests` prune 邏輯 | 這是 template 維護用測試；init 完成後，新 repo 的 `scripts/tests/` 只保留 [tests/test_release_notes.py](tests/test_release_notes.py) |

如果你要驗證腳本層級的回歸行為，優先從這個目錄的測試開始。

[返回開頭](#快速導覽)

## 常見用法

### 快速測試

```bash
# Bash / Git Bash
bash ./scripts/ci-test.sh
bash ./scripts/dev-test.sh -count=1
```

```powershell
# PowerShell
pwsh -File .\scripts\ci-test.ps1
pwsh -File .\scripts\dev-test.ps1 -count=1
```

### 安裝 Git hooks

```bash
bash ./scripts/install-git-hooks.sh
```

```powershell
pwsh -File .\scripts\install-git-hooks.ps1
```

### 產生 release notes 原始資料

```bash
python ./scripts/release-notes.py
python ./scripts/release-notes.py --format=markdown
```

### 執行腳本 regression tests

```bash
python -m unittest discover -s scripts/tests -p "test_*.py" -v
```

若想看整個 repo 的測試 workflow、pre-push gate 與 release 流程，請再搭配 [../README.md](../README.md) 一起看。

[返回開頭](#快速導覽)
