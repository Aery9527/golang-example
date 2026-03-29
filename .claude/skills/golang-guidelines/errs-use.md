# `internal/errs` 使用指南

本文件說明如何正確使用 `internal/errs` 模組回傳與處理錯誤。
更深入的設計細節（格式化輸出規格、stack 捕獲機制、duck-typing 架構等）→ 見 [docs/errs.md](../../../docs/errs.md)

## Error Code 常數（`pkg/errc`）

`pkg/errc` 定義了 `type Code string` 與專案統一的 error code 常數，格式為 `scope.category.detail`。
`Code` 提供便利方法直接生成 `*errs.Error`，不需每次手動傳入 code 字串。
若現有常數無法準確描述錯誤情境，應在 `pkg/errc/code.go` 中新增適當的 `Code` 常數，維持 `scope.category.detail` 格式：

```go
import "golan-example/pkg/errc"

errc.ServiceDBTimeout.New("connection deadline exceeded")
errc.ServiceDBTimeout.Wrap(dbErr, "query failed")
```

### Code 方法列表

| 方法 | 簽名 | 說明 |
|------|------|------|
| `New` | `(message string) *errs.Error` | 建立根錯誤 |
| `Newf` | `(format string, args ...any) *errs.Error` | 建立根錯誤（格式化 message） |
| `Wrap` | `(err error, message string) *errs.Error` | 包裝既有 error（nil 安全） |
| `Wrapf` | `(err error, format string, args ...any) *errs.Error` | 包裝既有 error（格式化 message，nil 安全） |

## 建構函式

若需要動態 code 或不使用預定義常數，可直接呼叫 `internal/errs` 的建構函式：

### 建立根錯誤（無 cause）

```go
err := errs.New("custom.code", "something went wrong")
err := errs.Newf("custom.code", "field %q must be positive", fieldName)
```

### 包裝既有錯誤（有 cause）

```go
err := errs.Wrap(dbErr, "custom.code", "query failed")
err := errs.Wrapf(dbErr, "custom.code", "query %s failed", tableName)
```

**Nil guard**：`Wrap(nil, ...)` / `Wrapf(nil, ...)` 安全回傳 `nil`，可直接寫：

```go
return errs.Wrap(err, "custom.code", "query failed") // err 為 nil 時回傳 nil
```

### 回傳型別

建構函式回傳 `*errs.Error`，但函式簽名一律用 `error` 介面：

```go
func LoadUser(id int) (*User, error) {
    row, err := db.Query(...)
    if err != nil {
        return nil, errc.ServiceDBTimeout.Wrap(err, "load user failed")
    }
    ...
}
```

## Error 型別方法

| 方法 | 回傳型別 | 說明 |
|------|---------|------|
| `Code()` | `string` | error code，例如 `"service.db.timeout"` |
| `Message()` | `string` | 原始 message，不含 `[CODE]` prefix |
| `Error()` | `string` | `[CODE] message` 格式 |
| `Unwrap()` | `error` | cause（支援 `errors.Is`/`errors.As` chain 走訪） |
| `StackTrace()` | `Stack` | 捕獲的 stack trace 防禦性複本（`[]Frame`） |
| `FormatStack()` | `string` | 格式化 stack 字串，每行 `Function (File:Line)` |

## 消費端使用模式

### 判斷特定 error code

```go
var target *errs.Error
if errors.As(err, &target) {
    switch target.Code() {
    case string(errc.ServiceDBTimeout):
        // 處理逾時
    case string(errc.ServiceDBConnection):
        // 處理連線失敗
    }
}
```

### 判斷特定 cause

```go
if errors.Is(err, sql.ErrNoRows) {
    // cause chain 中有 sql.ErrNoRows
}
```

### 傳給 log 系統

```go
logs.ErrorWith("query failed", func() (error, []any) {
    return err, []any{"table", "users"}
})
```

## 使用決策

| 情境 | 用法 |
|------|------|
| 新錯誤（預定義 code） | `errc.XxxYyy.New(msg)` 或 `errc.XxxYyy.Newf(fmt, args...)` |
| 包裝 error（預定義 code） | `errc.XxxYyy.Wrap(err, msg)` 或 `errc.XxxYyy.Wrapf(err, fmt, args...)` |
| 新錯誤（動態 code） | `errs.New(code, msg)` 或 `errs.Newf(code, fmt, args...)` |
| 包裝 error（動態 code） | `errs.Wrap(err, code, msg)` 或 `errs.Wrapf(err, code, fmt, args...)` |
| 取 code / message / stack | `errors.As(err, &target)` → `target.Code()` / `target.Message()` / `target.FormatStack()` |
| 判斷 cause chain | `errors.Is(err, sentinel)` |
| 結構化日誌 | 直接傳給 `logs.XxxWith()`，自動萃取 |
