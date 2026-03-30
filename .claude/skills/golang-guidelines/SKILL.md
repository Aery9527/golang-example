---
name: golang-guidelines
description: >-
  撰寫、修改或 review 任何 Go 程式碼時使用。載入 Go 撰寫守則（型別、錯誤處理、命名、Context、並發、測試等）。
---

你現在正在撰寫 Go 程式碼。請嚴格遵守以下守則，無論任務大小。

## 型別與介面

- 若 struct 明確實作介面，請在宣告上方加入 `var _ InterfaceName = (*StructName)(nil)` 做靜態驗證。
- 使用 `any`，不要使用 `interface{}`。
- 介面定義在**使用端**（consumer），而非實作端；介面盡量小，單一方法優先。

## 錯誤處理

- 錯誤需被處理或明確忽略（`_ = ...`），不可靜默丟棄。
- **一律使用 `internal/errs` 回傳錯誤**，不使用 `fmt.Errorf` 或 `errors.New`。
- **不自訂錯誤型別**——所有錯誤一律使用 `*errs.Error`，以 error code 區分類別。
- 函式回傳 error 時，型別一律寫 `error` 介面。
- 判斷錯誤用 `errors.Is` / `errors.As`，不直接比對字串。
- 不在 library 程式碼中使用 `panic`；`panic` 僅限程式初始化階段的不可回復錯誤。
- 完整 API 用法與使用模式 → 見 [errs-use.md](errs-use.md)

## 日誌

- **一律使用 `pkg/logs`（設定）與 `internal/logs`（引擎）記錄日誌**，不使用 `fmt.Println`、`log.Printf` 或外部 logging library（zap、logrus 等）。
- 完整 API 用法與使用模式 → 見 [logs-use.md](logs-use.md)

## 命名與可讀性

- 縮寫全大寫：`userID`、`httpClient`、`urlStr`，不寫 `userId`、`HttpClient`、`urlString`。
- 避免 naked return；有名稱的回傳值僅用於文件說明，不依賴其隱式 return。
- receiver 名稱使用型別首字母縮寫（1–2 字元），保持一致；不用 `self` / `this`。

## 文件化

- 所有 `struct` 宣告前都必須有註解，簡潔說明該型別的責任、使用情境與重要約束；若用途不直觀、初始化有前置條件、或使用流程超過一步，需補上簡短 usage example。
- 所有 `func` 宣告前都必須有註解，說明其目的、主要行為、重要副作用與關鍵輸入輸出；若行為不直觀、具副作用、會啟動並發、或呼叫方式容易誤用，需補上簡短 usage example。
- 若某個 struct func 是為了滿足另一個模組的 duck-typing 偵測而實作，註解中必須明確說明目標模組與對應介面，例如 `// FormatStack 供 pkg/log 的 stackProvider duck-typing 偵測使用`。
- 所有 `struct` field 都必須在欄位後方附上簡短註解，說明該欄位的意義或用途。
- 若 `struct` field 內容有固定格式（通常是 string 或 []string），註解中必須明確舉例，例如 `platform.key`、`NAME_AGE`、`["redis_a_b", "redis_c_d"]`、等。

## Context

- 接受 `context.Context` 的函式，`ctx` 必須是第一個參數。
- 不將 `context.Context` 儲存在 struct 欄位中；每次呼叫時傳入。

## 並發

- goroutine 啟動前，明確說明誰負責等待（`sync.WaitGroup`）或回收（`errgroup`）。
- channel 的方向在函式簽名中明確標示（`<-chan`、`chan<-`）。
- 共享資源的延遲初始化或狀態檢查，視情況使用 double-checked locking（先無鎖讀、再加鎖確認）避免極端併發下的重複執行或競態條件。
- 隨機數生成器（RNG）規劃要納入 thread-safety：**禁止在可能被多個 goroutine 共享的 struct 中持有 `math/rand.Rand` 實例**，因其內部狀態可變且非 thread-safe。
- 一般並發場景應優先使用 `math/rand/v2` 的全域函式（如 `rand.IntN()`、`rand.Float64()`）；其底層採用 per-thread ChaCha8，天然適合多 goroutine 併行呼叫。
- 若需要可重現的確定性隨機序列（例如測試），應建立獨立的 seeded RNG 實例，並確保該實例**不在 goroutine 間共享**；需要並行時，改為每個 goroutine 各自持有一份。
- 原則：任何可能被並行存取的物件內部，都不應持有非 thread-safe 的可變狀態；RNG、快取游標、暫存 buffer 等共享前都必須先確認其併發安全性。

## 測試

- 測試優先使用 `github.com/stretchr/testify/assert`；可行時補上或擴充測試覆蓋行為契約。
- 優先使用 table-driven tests（`[]struct{ name, input, want }`）；子測試用 `t.Run`。
- 每個套件必須同時包含**黑箱測試**（`package xxx_test`，驗證公開 API 契約與使用者視角行為）與**白箱測試**（`package xxx`，驗證內部邏輯、私有函式與邊界條件）。
- 以 `go test -coverprofile` 檢查覆蓋率，目標 **100%**；若特定路徑確實無法覆蓋（如 `runtime.Callers` 失敗等不可控路徑），須在測試檔案中以 `// coverage:ignore — <原因>` 明確標註理由。

## 其他

- `defer` 用於資源釋放（`Close`、`Unlock`）；在取得資源後立即 defer，不等到函式末尾。
- 零值應具備可用性（zero value usability）；struct 初始化後不需額外呼叫 `Init()` 才能使用。
- `init()` 僅用於確實必要的套件初始化，避免副作用難以追蹤；能用建構子解決的不用 `init()`。

## 模組參考（按需載入）

### 使用模組

當需要在程式碼中使用這些模組時，讀取對應的使用指南：

- 回傳或處理 error → [errs-use.md](errs-use.md)
- 記錄日誌 → [logs-use.md](logs-use.md)

### 修改模組

當需要修改模組本身的實作時，讀取完整設計文件了解模組全貌：

- 修改 `internal/errs` → [docs/errs.md](../../../docs/errs.md)
- 修改 `internal/logs` 或 `pkg/logs` → [docs/logs-design.md](../../../docs/logs-design.md)
