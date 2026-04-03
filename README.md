# golang-example

Go 專案模板（GitHub Template Repository），提供社群標準的專案結構。

## 快速導覽

- [快速開始](#快速開始)
- [初始化腳本說明](#初始化腳本說明)
- [初始化後的範例流程](#初始化後的範例流程)
- [專案結構](#專案結構)
- [開發測試與 Commit Workflow](#開發測試與-commit-workflow)
- [作為 Library 使用](#作為-library-使用)

## 快速開始

### 1. 從模板建立新專案

點擊 GitHub 上的 **「Use this template」** 按鈕建立新的 repository。

### 2. 初始化專案結構

```bash
# Linux / macOS / Git Bash
bash init.sh

# Windows PowerShell 7+
pwsh -File .\init.ps1

# Windows PowerShell 5.1
powershell.exe -File .\init.ps1
```

[返回開頭](#快速導覽)

## 初始化腳本說明

[init.sh](init.sh) 和 [init.ps1](init.ps1) 功能相同：

- 建立下述所有目錄與基礎 `.go` 檔案，已存在的檔案不會被覆蓋
- 生成 [cmd/app/main.go](cmd/app/main.go)，以 [internal/logs](internal/logs) 的 `Info` / `ErrorWith` 示範應用程式啟動、失敗與結束 logging
- 生成 [internal/handler/handler.go](internal/handler/handler.go)、[internal/service/service.go](internal/service/service.go)、[internal/repository/repository.go](internal/repository/repository.go) 的最小範例鏈路，示範 `error` 回傳慣例與 [pkg/errc](pkg/errc) 的錯誤建立/包裝方式
- 生成 `.gitignore`、`Makefile`、`Dockerfile`、`.env.example` 與必要的 `.gitkeep`
- 清空 `README.md`，並移除 `docs/superpowers`
- 初始化完成後，刪除位於 repo root 的 `init.sh` 與 `init.ps1`
- 若目前目錄是 Git repository，會自動執行 [scripts/install-git-hooks.sh](scripts/install-git-hooks.sh) 或 [scripts/install-git-hooks.ps1](scripts/install-git-hooks.ps1)；若不是，則顯示 `SKIP` 並繼續完成初始化

[返回開頭](#快速導覽)

## 初始化後的範例流程

初始化後的骨架不再只是 `Hello, World!`，而是直接帶出這個 repo 預設的 logging 與 error-handling 風格：

| 檔案 | 角色 | 產生的範例行為 |
| --- | --- | --- |
| [cmd/app/main.go](cmd/app/main.go) | 啟動入口 | 組裝 handler / service / repository，並以 `logs.Info` / `logs.ErrorWith` 記錄生命週期 |
| [internal/handler/handler.go](internal/handler/handler.go) | 邊界層 | 保持 `Handle() error` 介面，將錯誤交回上層統一處理 |
| [internal/service/service.go](internal/service/service.go) | 服務層 | 以 `errc.ServiceExampleRun.Wrap(err, "run example service")` 包裝下游錯誤 |
| [internal/repository/repository.go](internal/repository/repository.go) | 資料層 | 以 `errc.RepositoryExampleLoad.New("example repository is not implemented")` 建立根錯誤 |
| [pkg/errc/code.go](pkg/errc/code.go) | Error code 定義 | 提供 `ServiceExampleRun` 與 `RepositoryExampleLoad` 兩個 example 專用 code |

這樣初始化後的新專案，會直接留下可擴充的 handler → service → repository 骨架，以及 `logs` / `errs` / `errc` 的最小使用範例。

[返回開頭](#快速導覽)

## 專案結構

以下為 template repository 的主要結構；初始化完成後，repo root 的 [init.sh](init.sh) / [init.ps1](init.ps1) 會自刪，並留下生成後的專案骨架。

```
.
├── .editorconfig          # 編輯器設定（預設 LF；*.ps1 使用 UTF-8 BOM）
├── .githooks/             # repo-local Git hooks
│   └── pre-push
├── cmd/app/              # 應用程式進入點
│   └── main.go
├── internal/             # 私有程式碼（不可被外部 import）
│   ├── config/           # 應用程式設定
│   ├── errs/             # 錯誤處理（error code + stack trace + cause chain）
│   ├── handler/          # HTTP 處理器
│   ├── logs/             # 日誌引擎（Handler chain、Formatter、Sink）
│   ├── service/          # 商業邏輯層
│   └── repository/       # 資料存取層
├── pkg/                  # 可被外部 import 的共用套件
│   ├── errc/             # error code 與 *errs.Error 建立/包裝 helper
│   └── logs/             # 日誌公開 API（Configure DSL、re-exports）
├── api/                  # API 定義（OpenAPI、protobuf 等）
├── build/                # 建置與打包
│   └── Dockerfile
├── deployments/          # 部署設定（docker-compose、k8s 等）
├── docs/                 # 專案文件
│   ├── errs.md           # internal/errs 視覺化架構指南
│   └── logs-design.md    # 日誌模組設計規格
├── test/                 # 整合測試 / E2E 測試
│   └── integration/
├── scripts/              # 腳本工具
│   ├── ci-test.sh        # CI scope 快速測試（Bash）
│   ├── ci-test.ps1       # CI scope 快速測試（PowerShell）
│   ├── dev-test.sh       # 開發用完整 scoped 測試（Bash）
│   ├── dev-test.ps1      # 開發用完整 scoped 測試（PowerShell）
│   ├── go-test.sh        # shared Go test runner（Bash）
│   ├── go-test.ps1       # shared Go test runner（PowerShell）
│   ├── install-git-hooks.sh
│   ├── install-git-hooks.ps1
├── .env.example
├── .gitignore
├── init.ps1              # 結構初始化（PowerShell）
├── init.sh               # 結構初始化（Bash）
├── Makefile
└── go.mod
```

[返回開頭](#快速導覽)

## 開發測試與 Commit Workflow

### 安裝 repo-local Git hooks

先安裝 [scripts/install-git-hooks.sh](scripts/install-git-hooks.sh) 或 [scripts/install-git-hooks.ps1](scripts/install-git-hooks.ps1)，讓 Git 使用 repo 內的 [.githooks/pre-push](.githooks/pre-push)：

```bash
# Linux / macOS / Git Bash
bash scripts/install-git-hooks.sh

# Windows PowerShell
pwsh -File .\scripts\install-git-hooks.ps1
```

安裝完成後，local `git push` 會先經過 `pre-push`，並執行 scoped 的 `ci-test`。

### 編輯器設定

repo 內提供 [`.editorconfig`](.editorconfig)：

- 預設文字檔使用 `LF` 與 `UTF-8`
- `*.ps1` 額外指定為 `UTF-8 with BOM`
- 這個設定是為了避免含非 ASCII 文字的 PowerShell 腳本，在 Windows PowerShell 5.1 被錯誤解碼

若你需要修改 [init.ps1](init.ps1) 或其他需相容 Windows PowerShell 5.1 的 PowerShell 腳本，請保留這個 encoding 規則。

### 測試腳本

這套 workflow 目前只測兩個 package roots：

- `./internal/...`
- `./pkg/...`

若未來要納入其他 roots，可直接調整 shared runner 內的 root list。

#### `ci-test`

- [scripts/ci-test.sh](scripts/ci-test.sh)
- [scripts/ci-test.ps1](scripts/ci-test.ps1)

用途：

- 提供 push gate 使用的快速驗證
- 會附帶 `-short`
- 不產生 coverage
- 可額外轉發 `go test` 參數，例如 `-json`、`-run TestName`、`-count=1`

#### `dev-test`

- [scripts/dev-test.sh](scripts/dev-test.sh)
- [scripts/dev-test.ps1](scripts/dev-test.ps1)

用途：

- 提供開發期較完整的 scoped 測試
- 不附帶 `-short`
- 會產生 coverage artifacts
- 同樣支援額外 `go test` 參數

範例：

```bash
# Linux / macOS / Git Bash
bash scripts/ci-test.sh -json
bash scripts/dev-test.sh -count=1

# Windows PowerShell
pwsh -File .\scripts\ci-test.ps1 -json
pwsh -File .\scripts\dev-test.ps1 -count=1
```

#### `go-test`（shared runner，不直接呼叫）

- [scripts/go-test.sh](scripts/go-test.sh)
- [scripts/go-test.ps1](scripts/go-test.ps1)

`ci-test` 與 `dev-test` 的底層共用 runner，接收 `MODE` 參數（`ci` 或 `dev`）決定行為。一般不直接呼叫，由上層腳本代為傳入 mode。

主要職責：

- 根據 mode 組裝 `go test` 指令（`ci` 附加 `-short`；`dev` 附加 `-coverprofile`）
- 驗證 extra args，封鎖不合法用法（`-coverprofile`、`-args`、超出 `./internal/...` 與 `./pkg/...` 範圍的 `-coverpkg`）
- 將實際執行的指令寫入 `command.txt`，方便 debug 重放
- 將 exit code 寫入 `exit-code.txt`
- 處理跨平台路徑差異（WSL / Cygwin / native Windows）

### Test artifacts

每次執行都會刷新 `test-output/` 下對應模式的 artifacts；此目錄已加入 `.gitignore`，不會進版控。

- `test-output/ci-test/`
- `test-output/dev-test/`

常見 artifact：

| 檔案 | 說明 |
| --- | --- |
| `command.txt` | 實際執行的 `go test` command，可用來重放 |
| `exit-code.txt` | 該次執行的 exit code |
| `stdout.log` / `stdout.jsonl` | standard output |
| `stderr.log` | standard error |
| `coverage.out` | 僅 `dev-test` 產生的 coverage profile |
| `coverage-summary.txt` | 僅 `dev-test` 產生的 coverage 摘要 |

### Commit 與 push 規則

- local commit 預設不因為「只是要 commit」而先跑測試
- 若要手動驗證，可自行執行 `ci-test` 或 `dev-test`
- `git push` 前的驗證交給 [.githooks/pre-push](.githooks/pre-push)
- `pre-push` 失敗時，先看 `test-output/` 裡的 artifacts 再修正問題

若你使用支援 repo skills 的 agent，可參考 [.claude/skills/go-commit/SKILL.md](.claude/skills/go-commit/SKILL.md) 取得一致的 commit boundary 與 commit message workflow。

### Release Workflow

若要將 `develop` 發布到 `main`，請先保持 working tree 乾淨，確認 `develop` 已同步，並先執行 `ci-test`。之後可使用支援 repo skills 的 agent 觸發 `go-release`。

`go-release` 會：

- 驗證目前是正常的 `develop -> main` release，或辨識 `main` 上的 hotfix 路徑
- 執行 `python scripts/release-notes.py`，將上一個 tag 之後的 commit 收集到 `.tmp/release/raw-commits.json`
- 在乾淨 context 中濃縮 release notes，再根據確認後的 notes 建議下一個 `vMAJOR.MINOR.PATCH`
- 以 `git merge --no-ff develop -m "release: vX.Y.Z"` 將 release 併入 `main`，或在 hotfix 路徑直接於 `main` 建立 tag
- 建立 lightweight tag `vX.Y.Z`
- 完成後再詢問是否推送 `main` + tags，以及是否把 `main` merge 回 `develop`

`.tmp/` 為本地暫存 artifacts，已加入 `.gitignore`。

[返回開頭](#快速導覽)

## 作為 Library 使用

若將此 repo 作為依賴引入，可透過 `pkg/logs` 設定 logging 輸出行為。

```go
import "golan-example/pkg/logs"
```

`logs.Configure()` 以 `sync.Once` 保護，僅首次呼叫生效，適合在 `main()` 或程式初始化階段呼叫一次。

若想進一步了解如何設定 logs 模組，可參考 [`docs/logs-design.md` 的「pkg/logs 設定 API」章節](docs/logs-design.md#pkg/logs-設定-api)，裡面整理了 `Configure` 的預設值、設定 DSL 與合併規則。

**零設定（使用預設值）**

不呼叫 `Configure` 即可直接使用，預設行為為：全 level 啟用、Plain 格式、輸出至 Console（Debug/Info → stdout，Warn/Error → stderr）、帶 Caller 位置標記。

**自訂輸出**

```go
func main() {
    logs.Configure(
        // 全 level 輸出 JSON 到 stdout
        logs.Pipe(logs.JSON(), logs.ToStdout()),

        // Error level 額外寫入 rotating file
        logs.ForError(
            logs.Pipe(logs.JSON(), logs.ToFile("/var/log/app", logs.RotateConfig{})),
        ),
    )
}
```

**可用選項**

| 選項 | 說明 |
|---|---|
| `Pipe(formatter, output)` | 將指定 formatter 的輸出送往 output |
| `ForDebug / ForInfo / ForWarn / ForError` | 針對特定 level 套用獨立設定 |
| `NoCaller()` | 停用 caller 位置注入 |
| `NoInherit()` | level 設定不繼承全域設定 |
| `WithFilter(...)` | 加入 MessageFilter 或 KeyFilter |
| `WithEnrichment(...)` | 加入 Static enricher |

**Formatter**：`Plain()` / `JSON()`，可搭配 `WithTimeFormat(layout)` 調整時間格式。

**Output**：`Console()` / `Stdout()` / `Stderr()` / `ToWriter(w)` / `ToFile(basePath, ext, cfg)`。

[返回開頭](#快速導覽)
