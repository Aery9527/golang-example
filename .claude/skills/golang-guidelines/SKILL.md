---
name: golang-guidelines
description: >-
  撰寫、修改或 review 任何 Go 程式碼時使用。載入 Go 撰寫守則（型別、錯誤處理、命名、Context、並發、測試等）。
---

你現在正在撰寫 Go 程式碼。請嚴格遵守以下守則，無論任務大小。

## 型別與介面

1. 若 struct 明確實作介面，請在宣告上方加入 `var _ InterfaceName = (*StructName)(nil)` 做靜態驗證。
2. 使用 `any`，不要使用 `interface{}`。
3. 介面定義在**使用端**（consumer），而非實作端；介面盡量小，單一方法優先。

## 錯誤處理

4. 錯誤需被處理或明確忽略（`_ = ...`），不可靜默丟棄。
5. 使用 `fmt.Errorf("...: %w", err)` 包裝錯誤以保留 stack；判斷錯誤用 `errors.Is` / `errors.As`，不直接比對字串。
6. 自訂錯誤型別實作 `error` 介面時，套用規則 1 的靜態驗證。
7. 不在 library 程式碼中使用 `panic`；`panic` 僅限程式初始化階段的不可回復錯誤。

## 命名與可讀性

8. 縮寫全大寫：`userID`、`httpClient`、`urlStr`，不寫 `userId`、`HttpClient`、`urlString`。
9. 避免 naked return；有名稱的回傳值僅用於文件說明，不依賴其隱式 return。
10. receiver 名稱使用型別首字母縮寫（1–2 字元），保持一致；不用 `self` / `this`。

## 文件化

11. 所有 `struct` 宣告前都必須有註解，簡潔說明該型別的責任、使用情境與重要約束；若用途不直觀、初始化有前置條件、或使用流程超過一步，需補上簡短 usage example。
12. 所有 `func` 宣告前都必須有註解，說明其目的、主要行為、重要副作用與關鍵輸入輸出；若行為不直觀、具副作用、會啟動並發、或呼叫方式容易誤用，需補上簡短 usage example。
13. 所有 `struct` field 都必須在欄位後方附上簡短註解，說明該欄位的意義或用途。
14. 若 `struct` field 有固定格式、允許值集合或單位限制，註解中必須明確說明，並在適用時給一個具體例子，例如 `YYMMDD`、`20060102`、`https://api.example.com`、`ms`。

## Context

15. 接受 `context.Context` 的函式，`ctx` 必須是第一個參數。
16. 不將 `context.Context` 儲存在 struct 欄位中；每次呼叫時傳入。

## 並發

17. goroutine 啟動前，明確說明誰負責等待（`sync.WaitGroup`）或回收（`errgroup`）。
18. channel 的方向在函式簽名中明確標示（`<-chan`、`chan<-`）。

## 測試

19. 測試優先使用 `github.com/stretchr/testify/assert`；可行時補上或擴充測試覆蓋行為契約。
20. 優先使用 table-driven tests（`[]struct{ name, input, want }`）；子測試用 `t.Run`。
21. 每個套件必須同時包含**黑箱測試**（`package xxx_test`，驗證公開 API 契約與使用者視角行為）與**白箱測試**（`package xxx`，驗證內部邏輯、私有函式與邊界條件）。
22. 以 `go test -coverprofile` 檢查覆蓋率，目標 **100%**；若特定路徑確實無法覆蓋（如 `runtime.Callers` 失敗等不可控路徑），須在測試檔案中以 `// coverage:ignore — <原因>` 明確標註理由。

## 其他

23. `defer` 用於資源釋放（`Close`、`Unlock`）；在取得資源後立即 defer，不等到函式末尾。
24. 零值應具備可用性（zero value usability）；struct 初始化後不需額外呼叫 `Init()` 才能使用。
25. `init()` 僅用於確實必要的套件初始化，避免副作用難以追蹤；能用建構子解決的不用 `init()`。
