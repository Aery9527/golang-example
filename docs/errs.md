# `internal/errs` — 視覺化架構指南

## 快速導覽

- [Error 型別結構](#error-型別結構)
- [建立與包裝流程](#建立與包裝流程)
- [Error Chain 結構](#error-chain-結構)
- [Stack Trace 捕獲機制](#stack-trace-捕獲機制)
- [格式化輸出路由](#格式化輸出路由)
- [FormatStack 與 Duck-Typing](#formatstack-與-duck-typing)
- [跨模組零耦合架構](#跨模組零耦合架構)
- [使用情境決策樹](#使用情境決策樹)

---

## Error 型別結構

`Error` 結構體與其方法的完整面貌——包含實作的 interface 與對外提供的存取方式。

```mermaid
classDiagram
    class error {
        <<interface>>
        +Error() string
    }

    class fmtFormatter {
        <<interface>>
        +Format(f fmt.State, verb rune)
    }

    class Error {
        -code : string
        -message : string
        -cause : error
        -stack : Stack
        +Code() string
        +Message() string
        +Error() string
        +Unwrap() error
        +StackTrace() Stack
        +FormatStack() string
        +Format(f fmt.State, verb rune)
    }

    class Frame {
        +Function : string
        +File : string
        +Line : int
    }

    class Stack {
        <<type alias>>
        []Frame
    }

    error <|.. Error : 實作
    fmtFormatter <|.. Error : 實作
    Error *-- Stack : 持有
    Stack *-- Frame : 包含 0..N 個
```

> **設計要點**：`Error` 的所有欄位皆為 unexported，只透過方法存取；`StackTrace()` 回傳防禦性複本，外部修改不影響原始值。

[返回開頭](#快速導覽)

---

## 建立與包裝流程

四個建構函式的輸入輸出與 `nil` 安全行為。

```mermaid
flowchart LR
    subgraph 建立根錯誤
        New["New(code, msg)"]
        Newf["Newf(code, fmt, args...)"]
    end

    subgraph 包裝既有錯誤
        Wrap["Wrap(err, code, msg)"]
        Wrapf["Wrapf(err, code, fmt, args...)"]
    end

    New --> |"自動 capture stack"| E1["*Error\n（cause = nil）"]
    Newf --> |"fmt.Sprintf + capture"| E1

    Wrap --> NilCheck{err == nil?}
    Wrapf --> NilCheck

    NilCheck --> |"是"| Nil["回傳 nil"]
    NilCheck --> |"否"| E2["*Error\n（cause = err）"]
    E2 --> |"自動 capture stack"| E2
```

> **Nil Guard**：`Wrap(nil, ...)` / `Wrapf(nil, ...)` 安全回傳 `nil`，讓呼叫端可以直接 `return errs.Wrap(err, ...)` 而不需額外判斷。

[返回開頭](#快速導覽)

---

## Error Chain 結構

透過 `Wrap` 建立的 cause chain，與 stdlib `errors.Is` / `errors.As` 的走訪方式。

```mermaid
flowchart TD
    subgraph chain["Error Chain（由外到內）"]
        direction TB
        E1["🔴 *errs.Error\ncode: DB_QUERY_FAILED\nmessage: load user failed\nstack: ✅"]
        E2["🔴 *errs.Error\ncode: CONN_TIMEOUT\nmessage: connection pool exhausted\nstack: ✅"]
        E3["🟡 stdlib error\nsql: connection refused\nstack: ❌"]
    end

    E1 -->|"Unwrap()"| E2
    E2 -->|"Unwrap()"| E3
    E3 -->|"Unwrap()"| NIL["nil（chain 結束）"]

    IS["errors.Is(E1, target)"] -.->|"走訪整條 chain"| E1
    IS -.-> E2
    IS -.-> E3

    AS["errors.As(E1, &target)"] -.->|"找到第一個匹配型別"| E1
```

> **混合 chain**：chain 中可混合 `*errs.Error` 與一般 `error`。`%+v` 輸出時，`*errs.Error` 節點會印出 code + stack，一般 `error` 節點只印 message。

[返回開頭](#快速導覽)

---

## Stack Trace 捕獲機制

從呼叫端到最終存入 `Error.stack` 的完整路徑。

```mermaid
sequenceDiagram
    participant Caller as 呼叫端
    participant API as New / Wrap
    participant Capture as capture(skip=3)
    participant Runtime as runtime.Callers
    participant Frames as runtime.CallersFrames

    Caller ->> API: errs.New("CODE", "msg")
    API ->> Capture: capture(3)
    Capture ->> Runtime: Callers(3, pcs[:32])
    Runtime -->> Capture: n 個 program counter

    alt n == 0
        Capture -->> API: nil（無 stack）
    else n > 0
        Capture ->> Frames: CallersFrames(pcs[:n])
        loop 每個 frame
            Frames -->> Capture: {Function, File, Line}
            Note over Capture: File = filepath.Base(frame.File)
        end
        Capture -->> API: Stack（[]Frame）
    end

    API -->> Caller: *Error（含 stack）
```

> **skip = 3** 的含義：跳過 `runtime.Callers` → `capture` → `New`/`Wrap`，使第一個 frame 指向**呼叫端**的程式碼位置。

[返回開頭](#快速導覽)

---

## 格式化輸出路由

`fmt.Formatter` 實作中，不同 verb 的輸出路由與最終格式。

```mermaid
flowchart TD
    FMT["fmt.Printf(format, err)"]
    FMT --> Verb{verb?}

    Verb -->|"%s"| Simple["Error()\n→ [CODE] message"]
    Verb -->|"%v"| VFlag{有 '+' flag?}
    Verb -->|"%q"| Quoted["引號包裹\n→ &quot;[CODE] message&quot;"]

    VFlag -->|"否（%v）"| Simple
    VFlag -->|"是（%+v）"| Verbose["writeVerbose()"]

    Verbose --> Line1["[CODE] message"]
    Line1 --> StackOut["writeStack()\n→     at Func (File:Line)\n→     at Func (File:Line)"]
    StackOut --> HasCause{有 cause?}

    HasCause -->|"否"| Done["輸出結束"]
    HasCause -->|"是"| CauseType{cause 型別?}

    CauseType -->|"*errs.Error"| ErrsCause["Caused by: [CODE] message\n+ cause 的 stack frames"]
    CauseType -->|"一般 error"| PlainCause["Caused by: cause.Error()"]

    ErrsCause --> NextCause["繼續 Unwrap..."]
    PlainCause --> NextCause
    NextCause --> HasCause
```

> **Chain 走訪**：`%+v` 會沿著 `Unwrap()` 走訪整條 chain，每個節點根據型別決定輸出格式。

[返回開頭](#快速導覽)

---

## FormatStack 與 Duck-Typing

這是 `internal/errs` 最關鍵的設計決策之一——為什麼需要 `FormatStack()` 以及它如何實現零耦合。

### 問題：Go Interface 的型別耦合陷阱

```mermaid
flowchart LR
    subgraph problem["❌ 型別耦合問題"]
        direction TB
        Log1["pkg/log 定義介面：\nStackTracer { StackTrace() ??? }"]
        Errs1["internal/errs 的方法：\nStackTrace() errs.Stack"]
        Log1 -.->|"回傳型別必須完全一致"| Import["pkg/log 必須 import errs.Stack"]
        Import -->|"產生直接依賴"| Coupled["❌ 違反零耦合目標"]
    end

    subgraph solution["✅ FormatStack 解法"]
        direction TB
        Log2["pkg/log 定義介面：\nstackProvider { FormatStack() string }"]
        Errs2["internal/errs 的方法：\nFormatStack() string"]
        Log2 -.->|"回傳 stdlib string"| NoImport["無需 import 任何型別"]
        NoImport -->|"Go duck-typing 自動滿足"| Decoupled["✅ 零耦合達成"]
    end

    problem ~~~ solution
```

### 三層偵測策略

`pkg/log` 從 `error` 中萃取結構化資訊時，依序嘗試三層偵測——不需要 import `internal/errs`。

```mermaid
flowchart TD
    Start["收到 error"]
    Start --> T1{"實作 FormatStack 嗎？"}

    T1 -->|"是"| FS["🎯 Tier 1：取得 stack 字串\nerr.FormatStack&#40;&#41;"]
    FS --> Code{"實作 Code 嗎？"}

    T1 -->|"否"| T2{"實作 fmt.Formatter 嗎？"}

    T2 -->|"是"| FMT["📋 Tier 2：嘗試 %+v\nfmt.Sprintf 取得詳細輸出"]
    FMT --> Code

    T2 -->|"否"| Plain["📝 Tier 3：純文字\nerr.Error&#40;&#41;"]
    Plain --> Code

    Code -->|"是"| Full["結構化日誌\ncode + message + stack"]
    Code -->|"否"| Partial["部分結構化\nerror.text + stack"]

    style FS fill:#d4edda,stroke:#28a745,color:#155724
    style FMT fill:#fff3cd,stroke:#ffc107,color:#856404
    style Plain fill:#f8d7da,stroke:#dc3545,color:#721c24
```

### 為什麼不只用 `StackTrace()`？

| 方法 | 回傳型別 | duck-typing 可行性 | 適用場景 |
|------|----------|-------------------|----------|
| `StackTrace()` | `errs.Stack`（自定義型別） | ❌ 消費端必須 import `internal/errs` | 程式內部需要逐 frame 存取時 |
| `FormatStack()` | `string`（stdlib 型別） | ✅ 任何模組可定義相同簽名 | 日誌、監控等只需文字表示時 |

> **共存設計**：兩個方法並存——`StackTrace()` 給需要結構化存取的場景，`FormatStack()` 給跨模組零耦合的場景。

[返回開頭](#快速導覽)

---

## 跨模組零耦合架構

`internal/errs` 與未來 `pkg/log` 之間的依賴關係——透過 duck-typing interface 達成完全解耦。

```mermaid
flowchart TD
    subgraph errs_pkg["internal/errs"]
        direction TB
        ErrType["Error struct"]
        Methods["Code() string\nMessage() string\nFormatStack() string\nStackTrace() Stack"]
        ErrType --- Methods
    end

    subgraph log_pkg["pkg/log"]
        direction TB
        DuckInterfaces["私有 duck-typing 介面"]
        CP["codeProvider { Code() string }"]
        MP["messageProvider { Message() string }"]
        SP["stackProvider { FormatStack() string }"]
        DuckInterfaces --- CP
        DuckInterfaces --- MP
        DuckInterfaces --- SP
    end

    subgraph third_party["第三方 error library"]
        direction TB
        TPErr["CustomError"]
        TPMethods["FormatStack() string\n（相同簽名即可）"]
        TPErr --- TPMethods
    end

    Methods -.->|"Go duck-typing\n自動滿足"| DuckInterfaces
    TPMethods -.->|"Go duck-typing\n自動滿足"| DuckInterfaces

    errs_pkg <-->|"❌ 無 import 關係"| log_pkg
    third_party <-->|"❌ 無 import 關係"| log_pkg

    style errs_pkg fill:#e3f2fd,stroke:#1976d2,color:#0d47a1
    style log_pkg fill:#f3e5f5,stroke:#7b1fa2,color:#4a148c
    style third_party fill:#fff8e1,stroke:#f9a825,color:#e65100
```

> **開放性**：任何 error library 只要實作 `FormatStack() string`，就能自動被 `pkg/log` 偵測並萃取 stack 資訊——不需要任何 import 或 registry。

[返回開頭](#快速導覽)

---

## 使用情境決策樹

在不同場景下，應該使用哪個 API。

```mermaid
flowchart TD
    Start["需要回傳 error"]
    Start --> HasCause{"有 cause\n（原始 error）嗎？"}

    HasCause -->|"否"| NeedFmt{"message 需要格式化嗎？"}
    NeedFmt -->|"否"| UseNew["errs.New&#40;code, msg&#41;"]
    NeedFmt -->|"是"| UseNewf["errs.Newf&#40;code, fmt, args...&#41;"]

    HasCause -->|"是"| CauseFmt{"message 需要格式化嗎？"}
    CauseFmt -->|"否"| UseWrap["errs.Wrap&#40;err, code, msg&#41;"]
    CauseFmt -->|"是"| UseWrapf["errs.Wrapf&#40;err, code, fmt, args...&#41;"]

    UseNew --> Return["return err\n型別為 error 介面"]
    UseNewf --> Return
    UseWrap --> Return
    UseWrapf --> Return

    Return --> Consumer{"消費端如何處理？"}

    Consumer -->|"判斷 error 類型"| ErrAs["errors.As&#40;err, &amp;target&#41;\n取得 *errs.Error"]
    Consumer -->|"判斷特定 cause"| ErrIs["errors.Is&#40;err, sentinel&#41;\n走訪 chain"]
    Consumer -->|"印出詳細資訊"| PrintV["fmt.Printf&#40;&quot;%+v&quot;, err&#41;\nJava-style stack trace"]
    Consumer -->|"傳給 log 系統"| LogSys["log.Error&#40;msg, err&#41;\n自動 duck-typing 萃取"]

    style UseNew fill:#d4edda,stroke:#28a745,color:#155724
    style UseNewf fill:#d4edda,stroke:#28a745,color:#155724
    style UseWrap fill:#d4edda,stroke:#28a745,color:#155724
    style UseWrapf fill:#d4edda,stroke:#28a745,color:#155724
    style Return fill:#fff3cd,stroke:#ffc107,color:#856404
```

[返回開頭](#快速導覽)
