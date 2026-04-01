# golang-example

Go 專案模板（GitHub Template Repository），提供社群標準的專案結構。

## 快速導覽

- [快速開始](#快速開始)
- [初始化腳本說明](#初始化腳本說明)
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

# Windows PowerShell
pwsh init.ps1
```

[返回開頭](#快速導覽)

## 初始化腳本說明

[init.sh](init.sh) 和 [init.ps1](init.ps1) 功能相同：

- 建立下述所有目錄與基礎 `.go` 檔案，已存在的檔案不會被覆蓋
- 生成 `.gitignore`、`Makefile`、`Dockerfile`、`.env.example`
- 清空 `README.md`，並移除 `docs/superpowers`
- 初始化完成後，刪除位於 repo root 的 `init.sh` 與 `init.ps1`

[返回開頭](#快速導覽)

## 專案結構

```
.
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
│   ├── init.sh           # 結構初始化（Bash）
│   └── init.ps1          # 結構初始化（PowerShell）
├── .env.example
├── .gitignore
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
