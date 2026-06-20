// Package errorx 定义带业务码的错误。HTTP 层据此返回统一响应。
package errorx

import "fmt"

// Error 业务错误：业务码 + 用户可见消息 + HTTP 状态码 + 内部 cause（不暴露前端）。
type Error struct {
	Code    int32
	Message string
	HTTP    int
	cause   error
}

func (e *Error) Error() string {
	if e.cause != nil {
		return fmt.Sprintf("[%d] %s: %v", e.Code, e.Message, e.cause)
	}
	return fmt.Sprintf("[%d] %s", e.Code, e.Message)
}

func (e *Error) Unwrap() error { return e.cause }

// New 定义一个业务错误（通常在 codes.go 里集中声明）。
func New(code int32, message string, httpStatus int) *Error {
	return &Error{Code: code, Message: message, HTTP: httpStatus}
}

// Wrap 附加底层 cause（保留码/消息/HTTP，仅用于内部排查）。
func (e *Error) Wrap(cause error) *Error {
	return &Error{Code: e.Code, Message: e.Message, HTTP: e.HTTP, cause: cause}
}

// WithMsg 覆盖对外消息（码/HTTP 不变）。
func (e *Error) WithMsg(msg string) *Error {
	return &Error{Code: e.Code, Message: msg, HTTP: e.HTTP, cause: e.cause}
}

// FromError 把任意 error 归一成 *Error（未知错误 → 内部错误码）。
func FromError(err error) *Error {
	if err == nil {
		return nil
	}
	if e, ok := err.(*Error); ok {
		return e
	}
	return ErrInternal.Wrap(err)
}
