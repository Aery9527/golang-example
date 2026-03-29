package logs

// Handler 是 decorator chain 的單一節點。
// 不回傳 error——logging 系統內部消化所有失敗。
type Handler interface {
	Handle(entry Entry, next func(Entry))
}

// HandlerFunc 將普通函式轉為 Handler（便於測試與輕量場景）。
type HandlerFunc func(Entry, func(Entry))

func (f HandlerFunc) Handle(entry Entry, next func(Entry)) { f(entry, next) }

// SinkWriter 是 chain terminal 節點的介面——Sink 實作此介面。
type SinkWriter interface {
	Write(entry Entry)
}

// Chain 持有預組裝好的 closure chain，Execute 時是純 function call。
type Chain struct {
	exec func(Entry)
}

// NewChain 從 handlers + terminal function 組裝 closure chain。
// handlers 從前到後執行，最後呼叫 terminal（fan-out 到所有 Sink）。
func NewChain(handlers []Handler, terminal func(Entry)) *Chain {
	next := terminal
	for i := len(handlers) - 1; i >= 0; i-- {
		h := handlers[i]
		n := next
		next = func(e Entry) {
			h.Handle(e, n)
		}
	}
	return &Chain{exec: next}
}

// Execute 送 entry 進 chain。
func (c *Chain) Execute(entry Entry) {
	if c.exec != nil {
		c.exec(entry)
	}
}

// FanOut 建立 terminal function，將 entry fan-out 到多個 SinkWriter。
func FanOut(sinks []SinkWriter) func(Entry) {
	return func(entry Entry) {
		for _, s := range sinks {
			s.Write(entry)
		}
	}
}
