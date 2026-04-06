# `scripts/` 目錄說明

## 快速導覽

- [概覽](#概覽)
- [快速查詢](#快速查詢)
- [腳本詳解](#腳本詳解)
- [腳本測試](#腳本測試)
- [常見用法](#常見用法)

**依腳本快速跳轉**

- [ci-test.sh / ci-test.ps1](#ci-test)
- [dev-test.sh / dev-test.ps1](#dev-test)
- [go-test.sh / go-test.ps1](#go-test)
- [install-git-hooks.sh / install-git-hooks.ps1](#install-git-hooks)
- [link-agent-skills.sh / link-agent-skills.ps1](#link-agent-skills)
- [release-notes.py](#release-notes)

## 概覽

[`scripts/`](.) 收納這個 repo 的開發輔助腳本，主要分成四類：

1. 測試入口與共用 runner
2. Git hooks 與 agent skills 輔助工具
3. release notes 資料收集器
4. 腳本本身的 regression tests

如果你只想知道「現在該跑哪支」，先看 [快速查詢](#快速查詢)。如果你要操作 `.agents/skills` 與 [`.claude/skills`](../.claude/skills) 的連結模式、`.gitignore` 同步與清理行為，直接跳到 [link-agent-skills](#link-agent-skills)。

[返回開頭](#快速導覽)

## 快速查詢

| 腳本 | 平台 / 類型 | 主要用途 | 詳細說明 |
| --- | --- | --- | --- |
| [ci-test.sh](ci-test.sh) / [ci-test.ps1](ci-test.ps1) | Bash / PowerShell | 跑 pre-push 同級的快速 scoped tests | [ci-test](#ci-test) |
| [dev-test.sh](dev-test.sh) / [dev-test.ps1](dev-test.ps1) | Bash / PowerShell | 跑開發期較完整的 scoped tests 與 coverage | [dev-test](#dev-test) |
| [go-test.sh](go-test.sh) / [go-test.ps1](go-test.ps1) | Bash / PowerShell | 真正組裝 `go test`、寫 artifacts、驗證參數 | [go-test](#go-test) |
| [install-git-hooks.sh](install-git-hooks.sh) / [install-git-hooks.ps1](install-git-hooks.ps1) | Bash / PowerShell | 安裝 repo-local Git hooks，設定 `core.hooksPath=.githooks` | [install-git-hooks](#install-git-hooks) |
| [link-agent-skills.sh](link-agent-skills.sh) / [link-agent-skills.ps1](link-agent-skills.ps1) | Bash / PowerShell | 將 `.agents/skills` 對齊到 [`.claude/skills`](../.claude/skills) | [link-agent-skills](#link-agent-skills) |
| [release-notes.py](release-notes.py) | Python | 收集 Conventional Commits，輸出 release notes 原始資料或 Markdown | [release-notes](#release-notes) |
| [tests/test_release_notes.py](tests/test_release_notes.py) | Python test | 驗證 [release-notes.py](release-notes.py) 的分類與輸出格式 | [腳本測試](#腳本測試) |

[返回開頭](#快速導覽)

## 腳本詳解

### ci-test

相關檔案：[ci-test.sh](ci-test.sh)、[ci-test.ps1](ci-test.ps1)

`ci-test` 是最薄的一層入口，固定用 `ci` mode 呼叫 [go-test.sh](go-test.sh) / [go-test.ps1](go-test.ps1)：

- 測試目標固定為 `./internal/...` 與 `./pkg/...`
- 會附加 `-short`
- 不產生 coverage artifacts
- 適合 pre-push gate 與快速 smoke check

### dev-test

相關檔案：[dev-test.sh](dev-test.sh)、[dev-test.ps1](dev-test.ps1)

`dev-test` 也是 [go-test.sh](go-test.sh) / [go-test.ps1](go-test.ps1) 的包裝入口，但使用 `dev` mode：

- 不附加 `-short`
- 會產生 coverage profile 與 coverage summary
- 同樣只測 `./internal/...` 與 `./pkg/...`
- 適合開發中想多看一層 coverage 與較完整輸出時使用

### go-test

相關檔案：[go-test.sh](go-test.sh)、[go-test.ps1](go-test.ps1)

`go-test` 是共用 runner 本體，負責真正的測試 orchestration：

- 清空並重建 `test-output/<mode>-test/`
- 固定測試 scope 為 `./internal/...` 與 `./pkg/...`
- 驗證 extra args，避免不合法用法，例如 `-args`、自訂 `-coverprofile` 或超出 scope 的 `-coverpkg`
- 將實際執行命令寫入 `command.txt`
- 永遠保留 `exit-code.txt` 與 `stderr.log`
- 依 `-json` 決定將 standard output 寫入 `stdout.log` 或 `stdout.jsonl`
- `dev` mode 額外產生 `coverage.out` 與 `coverage-summary.txt`

日常開發通常直接跑 [ci-test.sh](ci-test.sh) / [ci-test.ps1](ci-test.ps1) 或 [dev-test.sh](dev-test.sh) / [dev-test.ps1](dev-test.ps1) 就夠了，不需要直接手敲 `go-test`。

### install-git-hooks

相關檔案：[install-git-hooks.sh](install-git-hooks.sh)、[install-git-hooks.ps1](install-git-hooks.ps1)

這兩支腳本的責任很單純：

- 確認目前在 Git repository 內
- 設定 local Git config：`core.hooksPath=.githooks`

完成後，`git push` 會走 [../.githooks/pre-push](../.githooks/pre-push)，也就是 repo 自己維護的 pre-push gate。

### link-agent-skills

相關檔案：[link-agent-skills.sh](link-agent-skills.sh)、[link-agent-skills.ps1](link-agent-skills.ps1)

`link-agent-skills` 讓 `.agents/skills` 與 [`.claude/skills`](../.claude/skills) 維持一致，同時保留不同工具鏈對 skills 路徑命名的使用習慣。兩支腳本都會根據自己的檔案位置定位 repo root，所以只要用正確路徑呼叫，從 repo 任意目錄執行都可以。

這兩支腳本都支援：

- 互動式選單
- 自動同步 [`.gitignore`](../.gitignore) 條目
- 清理失效的 skill 連結
- 單一整目錄連結或逐 skill 連結

執行方式：

```bash
bash ./scripts/link-agent-skills.sh
```

```powershell
pwsh -File .\scripts\link-agent-skills.ps1
```

連結模式如下：

| 模式 | Bash 行為 | PowerShell 行為 | 適合情境 |
| --- | --- | --- | --- |
| `0` | 取消 | 取消 | 只想離開選單 |
| `1` | 將 `.agents/skills` 建成單一 symlink 指向 [`.claude/skills`](../.claude/skills) | 將 `.agents/skills` 建成單一 junction 指向 [`.claude/skills`](../.claude/skills) | 想讓 `.agents/skills` 完全鏡像來源目錄 |
| `2` | 在 `.agents/skills/` 下逐一建立 per-skill symlink | 在 `.agents/skills/` 下逐一建立 per-skill junction | 想保留 `.agents/skills/` 目錄本體，只同步各 skill |
| `3` | 移除腳本建立的整體 symlink 或 per-skill symlink，並清理 [`.gitignore`](../.gitignore) | 移除腳本建立的整體 junction 或 per-skill junction，並清理 [`.gitignore`](../.gitignore) | 想回復未連結狀態 |

模式選擇建議：

- 想要 `.agents/skills/` 完全鏡像 [`.claude/skills`](../.claude/skills)：選 **Mode 1**
- 想保留 `.agents/skills/` 目錄本體，只對每個 skill 建立個別連結：選 **Mode 2**
- 想回復未連結狀態並清掉腳本建立的 ignore 條目：選 **Mode 3**

`.gitignore` 行為：

- **Mode 1 / Mode 2**：自動加入對應的 `.agents/skills` 路徑
- **Mode 3**：移除腳本建立的對應條目
- 已存在的條目不會重複寫入
- Mode 2 若遇到「已存在但不是連結」的目標，會保留原物件並略過該 skill

### release-notes

相關檔案：[release-notes.py](release-notes.py)

`release-notes.py` 是 release workflow 的資料收集器。它會讀 Git history，挑出 Conventional Commits，並做以下事情：

- 自動解析上一個 tag，或接受手動指定的 revision range
- 忽略 `Merge ...`、`merge:`、`release:` 這類 release noise
- 依 `feat`、`fix`、`perf`、`docs` 等群組分類 commit
- 擷取 `BREAKING CHANGE:` footer
- 預設將結果寫到 `.tmp/release/raw-commits.json`
- 使用 `--format=markdown` 時，改為直接輸出可讀的 Markdown release notes

這支腳本主要提供給 release workflow 與 agent skill 使用，但也可以獨立手動執行。

[返回開頭](#快速導覽)

## 腳本測試

[`tests/`](tests) 在初始化完成後會保留下列腳本層級 `unittest`：

| 檔案 | 主要用途 | 備註 |
| --- | --- | --- |
| [tests/test_release_notes.py](tests/test_release_notes.py) | 驗證 [release-notes.py](release-notes.py) 在暫存 Git repo 中的分類、tag range、breaking change 與 Markdown 輸出行為 | 這支測試會保留在 init 後的新 repo |

如果你要驗證初始化後 repo 會保留的腳本測試內容，這裡是最直接的入口。

[返回開頭](#快速導覽)

## 常見用法

### 快速測試

```bash
bash ./scripts/ci-test.sh
bash ./scripts/dev-test.sh -count=1
```

```powershell
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

### 連結 agent skills

```bash
bash ./scripts/link-agent-skills.sh
```

```powershell
pwsh -File .\scripts\link-agent-skills.ps1
```

### 產生 release notes

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
