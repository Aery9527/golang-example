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

### 3. 執行

```bash
go run ./cmd/app
```

## 專案結構

```
.
├── cmd/app/              # 應用程式進入點
│   └── main.go
├── internal/             # 私有程式碼（不可被外部 import）
│   ├── config/           # 應用程式設定
│   ├── handler/          # HTTP 處理器
│   ├── logs/             # 日誌引擎（Handler chain、Formatter、Sink）
│   ├── service/          # 商業邏輯層
│   └── repository/       # 資料存取層
├── pkg/                  # 可被外部 import 的共用套件
│   ├── errs/             # 錯誤處理（error code + stack trace + cause chain）
│   │   ├── errs.go       # Error 型別、New/Newf 建構與 fmt.Formatter 實作
│   │   ├── stack.go      # Frame/Stack 型別與 call stack 捕獲
│   │   └── wrap.go       # Wrap/Wrapf 包裝既有 error
│   └── logs/             # 日誌公開 API（Configure DSL、re-exports）
├── api/                  # API 定義（OpenAPI、protobuf 等）
├── build/                # 建置與打包
│   └── Dockerfile
├── deployments/          # 部署設定（docker-compose、k8s 等）
├── docs/                 # 專案文件
│   ├── errs.md           # pkg/errs 視覺化架構指南
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
