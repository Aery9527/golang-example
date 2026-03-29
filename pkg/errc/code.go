package errc

import "golan-example/internal/errs"

// Code 代表應用層 error code，格式為 scope.category.detail（例如
// "service.db.timeout"）。提供便利方法直接生成 *errs.Error，省去每次傳入 code
// 字串的重複：
//
//	return errc.ServiceDBTimeout.New("connection deadline exceeded")
//	return errc.ServiceDBTimeout.Wrap(err, "query failed")
type Code string

// New 以此 Code 與 message 建立根 Error。Stack trace 從呼叫者的角度捕獲。
func (c Code) New(message string) *errs.Error {
	return errs.NewWithSkip(1, string(c), message)
}

// Newf 以此 Code 與格式化 message 建立根 Error。Stack trace 從呼叫者的角度捕獲。
func (c Code) Newf(format string, args ...any) *errs.Error {
	return errs.NewfWithSkip(1, string(c), format, args...)
}

// Wrap 以此 Code 包裝既有 error。若 err 為 nil，回傳 nil。
func (c Code) Wrap(err error, message string) *errs.Error {
	return errs.WrapWithSkip(1, err, string(c), message)
}

// Wrapf 以此 Code 包裝既有 error 並格式化 message。若 err 為 nil，回傳 nil。
func (c Code) Wrapf(err error, format string, args ...any) *errs.Error {
	return errs.WrapfWithSkip(1, err, string(c), format, args...)
}

// Error code 常數。
const (
	// service — 業務服務層錯誤
	ServiceDBTimeout    Code = "service.db.timeout"    // 資料庫操作逾時
	ServiceDBConnection Code = "service.db.connection"  // 資料庫連線失敗

	// internal — 內部不可預期錯誤
	InternalUnknown Code = "internal.unknown" // 未分類的內部錯誤
)
