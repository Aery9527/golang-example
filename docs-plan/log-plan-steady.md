更新 [log-plan.md](log-plan.md) 加入或調整為以下內容

- 模組名稱為 `logs`, 並且為 internal package, 但 pkg 要提供外部可以設定的入口, 例如可以設定 level, format, output 等

- 提供基礎的 `logs.Debug()`, `logs.Info()`, `logs.Warn()`, `logs.Error()` 等 func, 但具體是橋接到實體的 `var struct`,
  方便上述提到的設定可以抽換實作, 4 種 level 分別對應 4 個 `var struct` 以支援完全不同的輸出設定

- 具體類似 java slf4j 那樣可以組合設定,
  我目前想到的是 format(plat or json or both) 跟 output(console or file or both) 兩個維度的組合,
  如果你有想到其他的維度也請提出, 因此 format 跟 output 必須要有良好的架構支援擴充
 
- 所有 log func 都 **必須** **只能** 是 lazy 操作特性, 因為可以避免 level disable 時產生不必要的參數 slice,
  這點是我考量系統運作上的效能問題, 如有其他想法也請提出來討論, 但我認為這點是非常重要

- 所有 log func(debug, info, warn, error),
  都要可以接受 "msg + kv-part" 跟 "msg + error",
  對於 error 的處理要如 [errs.md](../docs/errs.md) 內提到的作法,
  error 都要放在最後面輸出, 拿的到 stack trace 的話就輸出 stack trace, 拿不到的話就輸出 error.Error() 的內容

- kv-part 的部分, 類似 `logs.Info(msg, "key1", value1, "key2", value2, ...)`
  - 在 plat 輸出時每一組 kv-part 就是一行 所以預期輸出是:
    ```
    yymmdd HH:mm:ss.SSS [format(%-5s, LEVEL)] msg
      [1] key1 : value1
      [2] key1 : value2
      ...
    ```
    如果參數超過2位數則使用 [01], [02], ...
    也就是說會優先判斷多少個參數量再決定 `%0?d`
  - json 輸出時則是把 kv-part 直接放在 json 裡面, 有任何衝突的 key 不可覆蓋, 而是額外加 prefix 來區別, 同時在內部額外寫一個 warn log 來紀錄這件事
  - json 輸出有 error 時, 需要有 `err_code`, `err_msg`, `err_stack` 等欄位, 若 error 拿不到這些東西則 value 為 null, 但 key 必須存在
  - json 輸出時 key 一定要是 string, value 如果一般狀態(string, number)或 json.Marshal-able 或 json.RawMessage 就直接輸出,
    其他類型則使用 fmt.Sprintf("%+v") 的內容來輸出, 尤其是數字一定要維持原本的數字型態來輸出, 因為 json 可以方便拿來做統計
  - args 不為偶數時直接使用參數 index 當作 key, 並且內部額外寫一個 warn log 來紀錄這件事, 用意是為了避免使用者在使用上出錯卻沒有察覺到

- 支援 args binding 的功能, 例如 `logs.With("key1", value1, "key2", value2))` 然後回傳一個新的 logger instance,
  這樣就可以在後續的 log func 裡面自動帶入這些 kv-part 的參數, 這樣就不需要每次都重複寫一樣的 kv-part 了,
  同時也可以支援多層的 bind args, 例如 `logs.With("key1", value1).With("key2", value2)` 等等
