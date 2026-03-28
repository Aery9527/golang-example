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

## Context

11. 接受 `context.Context` 的函式，`ctx` 必須是第一個參數。
12. 不將 `context.Context` 儲存在 struct 欄位中；每次呼叫時傳入。

## 並發

13. goroutine 啟動前，明確說明誰負責等待（`sync.WaitGroup`）或回收（`errgroup`）。
14. channel 的方向在函式簽名中明確標示（`<-chan`、`chan<-`）。

## 測試

15. 測試優先使用 `github.com/stretchr/testify/assert`；可行時補上或擴充測試覆蓋行為契約。
16. 優先使用 table-driven tests（`[]struct{ name, input, want }`）；子測試用 `t.Run`。
17. 測試檔案中可使用 `_test` package（black-box testing）驗證公開 API 行為。

## 其他

18. `defer` 用於資源釋放（`Close`、`Unlock`）；在取得資源後立即 defer，不等到函式末尾。
19. 零值應具備可用性（zero value usability）；struct 初始化後不需額外呼叫 `Init()` 才能使用。
20. `init()` 僅用於確實必要的套件初始化，避免副作用難以追蹤；能用建構子解決的不用 `init()`。
