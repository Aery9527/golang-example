# `pkg/logs` + `internal/logs` 使用指南

本文件說明如何正確使用日誌模組記錄 log 與設定輸出行為。
更深入的設計細節（Handler Chain 機制、Formatter 格式規格、duck-typing 偵測、internal warn 等）→ 見 [docs/logs-design.md](../../../docs/logs-design.md)

## 快速上手

零設定即可用——不呼叫 `Configure()` 等同於：全 level 啟用、Plain 格式、Console 輸出（Debug+Info → stdout，Warn+Error → stderr）、帶 Caller 位置。

```go
import "golan-example/internal/logs"

logs.Info("server started", nil) // 純 message，無 kv-pairs
```

## Logger API

### 基本 log（帶 kv-pairs）

```go
logs.Info("user login", func() []any {
    return []any{"user_id", uid, "ip", remoteAddr}
})
```

- 第二個參數是 **lazy closure**——level 未啟用時 closure 不執行，zero allocation。
- 傳 `nil` 代表無 kv-pairs：`logs.Info("msg", nil)`

### 帶 error 的 log

```go
logs.ErrorWith("query failed", func() (error, []any) {
    return err, []any{"table", "users", "id", userID}
})
```

`XxxWith` variant 的 closure 回傳 `(error, []any)`。

### 完整方法列表

| 方法 | 簽名 |
|------|------|
| `Debug` / `Info` / `Warn` / `Error` | `(msg string, fn func() []any)` |
| `DebugWith` / `InfoWith` / `WarnWith` / `ErrorWith` | `(msg string, fn func() (error, []any))` |
| `With` | `(args ...any) *Logger` |

以上皆有 package-level 便利函式（透過 `defaultLogger` 委派）與 `*Logger` 方法兩個版本。

### With() 綁定 context

```go
l := logs.With("request_id", rid, "method", "GET")
l.Info("handling request", nil)
l.ErrorWith("failed", func() (error, []any) { return err, nil })
```

- `With()` 回傳**新** Logger，共享 chain，各自持有 bound kv-pairs。
- 支援鏈式呼叫：`logs.With("a", 1).With("b", 2)`

## Configure DSL（`pkg/logs`）

```go
import "golan-example/pkg/logs"

logs.Configure(opts ...Option) // sync.Once 保護，僅首次呼叫生效
```

### 全局預設值

| 項目 | 預設值 |
|------|--------|
| Level | 全部 4 個 level 啟用 |
| Formatter | `Plain()` |
| Output | `ToConsole()` |
| Enrichment | `Caller()` |

### 設定範例

**零設定（使用預設值）**

不呼叫 `Configure()` 即可直接使用。

**全局 JSON 輸出到 stdout**

```go
logs.Configure(
    logs.Pipe(logs.JSON(), logs.ToStdout()),
)
```

**多 output + per-level 設定**

```go
logs.Configure(
    // 全局：Plain 到 Console
    logs.Pipe(logs.Plain(), logs.ToConsole()),

    // Error 額外寫 JSON 到 rotating file
    logs.ForError(
        logs.Pipe(logs.JSON(), logs.ToFile("error", logs.RotateConfig{})),
        logs.WithEnrichment(logs.Static("service", "api")),
    ),

    // Debug 不要 caller
    logs.ForDebug(
        logs.NoCaller(),
    ),
)
```

**過濾特定 message**

```go
logs.Configure(
    logs.WithFilter(logs.FilterByMessage(func(m string) bool {
        return m != "heartbeat"
    })),
    logs.Pipe(logs.Plain(), logs.ToConsole()),
)
```

### Option 完整列表

| Option | 說明 |
|--------|------|
| `Pipe(formatter, output)` | 建立 formatter + output 組合（可多次呼叫，fan-out） |
| `WithFilter(filters...)` | 加入 filter（多個 filter 為 AND 關係） |
| `WithEnrichment(enrichers...)` | 加入 enricher |
| `NoCaller()` | 停用預設 Caller enrichment |
| `NoInherit()` | 不繼承全局設定（僅在 `ForXxx` 內使用） |
| `ForDebug(opts...)` / `ForInfo(opts...)` / `ForWarn(opts...)` / `ForError(opts...)` | per-level 追加設定 |

### 合併規則

- 全局設定作為 base，每個 level 繼承。
- `ForXxx()` 的設定**追加**到全局之上（不是覆寫）。
- `Caller()` 預設啟用；用 `NoCaller()` 明確移除。
- `NoInherit()` 讓該 level 完全不繼承全局設定。

## Formatter / Output / Filter / Enrichment

### Formatter

| 建構函式 | 格式 |
|---------|------|
| `Plain(opts...)` | 人類可讀，帶編號 kv-pairs |
| `JSON(opts...)` | 結構化 JSON |

自訂時間格式：`logs.Plain(logs.WithTimeFormat(time.RFC3339))`

### Output

| 建構函式 | 行為 |
|---------|------|
| `ToConsole()` | Debug+Info → stdout，Warn+Error → stderr |
| `ToStdout()` | 一律 stdout |
| `ToStderr()` | 一律 stderr |
| `ToWriter(w io.Writer)` | 一律寫入 w |
| `ToFile(name, cfg)` | RotatingFileWriter，帶 rotation |

### Filter

```go
logs.FilterByMessage(func(msg string) bool { return msg != "heartbeat" })
logs.FilterByKey("env", func(v string) bool { return v == "prod" })
```

多個 filter 串接為 AND 關係。

### Enrichment

| 建構函式 | 說明 |
|---------|------|
| `Caller()` | 注入 `"caller"` kv-pair（預設啟用） |
| `Static(key, val)` | 注入固定 kv-pair |
| `NoCaller()` | 移除預設 Caller |

### RotateConfig

```go
logs.ToFile("app", logs.RotateConfig{
    MaxSize:    100 * 1024 * 1024, // 100 MB（預設）
    MaxBackups: 10,                // 保留 10 個舊檔（預設）
    MaxAge:     7 * 24 * time.Hour, // 7 天（預設）
    Compress:   true,              // gzip 壓縮舊檔
})
```
