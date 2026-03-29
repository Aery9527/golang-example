# golang-example

Go 專案模板（GitHub Template Repository），提供社群標準的專案結構。

## 快速開始

### 1. 從模板建立新專案

點擊 GitHub 上的 **「Use this template」** 按鈕建立新的 repository。

### 2. 初始化專案結構

```bash
# Linux / macOS / Git Bash
bash scripts/init.sh

# Windows PowerShell
pwsh scripts/init.ps1
```

## 專案結構

```
.
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
│   ├── init.sh           # 結構初始化（Bash）
│   └── init.ps1          # 結構初始化（PowerShell）
├── .env.example
├── .gitignore
├── Makefile
└── go.mod
```

## 作為 Library 使用

若將此 repo 作為依賴引入，可透過 `pkg/logs` 設定 logging 輸出行為。

```go
import "golan-example/pkg/logs"
```

`logs.Configure()` 以 `sync.Once` 保護，僅首次呼叫生效，適合在 `main()` 或程式初始化階段呼叫一次。

**零設定（使用預設值）**

不呼叫 `Configure` 即可直接使用，預設行為為：全 level 啟用、Plain 格式、輸出至 Console（Debug/Info → stdout，Warn/Error → stderr）、帶 Caller 位置標記。

**自訂輸出**

```go
func main() {
    logs.Configure(
        // 全 level 輸出 JSON 到 stdout
        logs.Pipe(logs.JSON(), logs.Stdout()),

        // Error level 額外寫入 rotating file
        logs.ForError(
            logs.Pipe(logs.JSON(), logs.ToFile("/var/log/app", ".log", logs.RotateConfig{})),
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

## Makefile 指令

| 指令 | 說明 |
|---|---|
| `make build` | 編譯至 `./bin/app` |
| `make run` | 直接執行 `cmd/app` |
| `make test` | 執行所有測試 |
| `make lint` | 執行 golangci-lint 靜態分析 |
| `make clean` | 清除編譯產物 |

## 初始化腳本說明

`scripts/init.sh` 和 `scripts/init.ps1` 功能相同：

- 依據 `go.mod` 讀取模組名稱
- 建立上述所有目錄與基礎 `.go` 檔案
- 生成 `.gitignore`、`Makefile`、`Dockerfile`、`.env.example`
- **冪等執行** — 已存在的檔案不會被覆蓋，可安全重複執行
