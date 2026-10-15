package ohmyhttp

import (
	"errors"
	"fmt"
)

// 本文件定义进入 HTTP 响应的错误。怎么把它们写成响应，见 error_mapper.go。
//
// 客户端看到的是状态码和 Problem。除了实现 HTTPError 的错误，以及媒体类型错误
// 自己的 Error() 文本，普通 error 的 Error() 默认不会写进响应。
//
// 普通错误不在这里保存调用栈。PanicError 保存 Recovery 得到的堆栈，供观察器读取。
// 业务代码要在返回处留下调用栈时，可使用 ohmyhttp/errorx。

// Problem 是默认的 JSON 错误结构。
type Problem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// HTTPError 由自行声明 HTTP 状态码、错误码和客户端说明的错误实现。
// 默认映射用 HTTPStatus、ErrorCode、ErrorMessage 和 ErrorDetails 填写 Problem。
// 状态码必须在 400 到 599 之间，否则改为 500。
type HTTPError interface {
	error
	HTTPStatus() int
	ErrorCode() string
	ErrorMessage() string
	ErrorDetails() any
}

// Error 是 HTTPError 的默认实现。
type Error struct {
	status  int
	code    string
	message string
	details any
}

var _ HTTPError = (*Error)(nil)

// NewError 创建分别包含 HTTP 状态码和应用错误码的错误。
func NewError(status int, code string, message string) *Error {
	return &Error{status: status, code: code, message: message}
}

// WithDetails 返回包含结构化公开详情的浅拷贝。
func (e *Error) WithDetails(details any) *Error {
	copy := *e
	copy.details = details
	return &copy
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) HTTPStatus() int      { return e.status }
func (e *Error) ErrorCode() string    { return e.code }
func (e *Error) ErrorMessage() string { return e.message }
func (e *Error) ErrorDetails() any    { return e.details }

// RequestBindingError 标记 HTTP 请求绑定过程中产生的错误。
type RequestBindingError struct {
	Cause error
}

var _ error = (*RequestBindingError)(nil)

func (e *RequestBindingError) Error() string { return "bind request: " + e.Cause.Error() }
func (e *RequestBindingError) Unwrap() error { return e.Cause }

// UnsupportedMediaTypeError 表示请求体没有可用的解码器。
type UnsupportedMediaTypeError struct {
	MediaType string
}

var _ error = (*UnsupportedMediaTypeError)(nil)

func (e *UnsupportedMediaTypeError) Error() string {
	if e.MediaType == "" {
		return "request Content-Type is required"
	}
	return fmt.Sprintf("unsupported Content-Type %q", e.MediaType)
}

// ValidationError 标记请求自动校验过程中产生的错误。
type ValidationError struct {
	Cause   error
	Details any
}

var _ error = (*ValidationError)(nil)

// NewValidationError 创建带有公开校验详情的请求校验错误。
func NewValidationError(cause error, details any) *ValidationError {
	if cause == nil {
		cause = errors.New("request validation failed")
	}
	return &ValidationError{Cause: cause, Details: details}
}

func (e *ValidationError) Error() string {
	if e.Cause == nil {
		return "validate request"
	}
	return "validate request: " + e.Cause.Error()
}

func (e *ValidationError) Unwrap() error { return e.Cause }

// PanicError 为错误观察器保留 recover 得到的值和调用堆栈。
type PanicError struct {
	Value any
	Stack []byte
}

var _ error = (*PanicError)(nil)

func (e *PanicError) Error() string { return fmt.Sprintf("panic recovered: %v", e.Value) }

func (e *PanicError) StackTrace() string { return string(e.Stack) }
