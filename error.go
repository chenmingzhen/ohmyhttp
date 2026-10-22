package ohmyhttp

import (
	"errors"
	"fmt"
	"net/http"
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

// ErrorOption 设置 NewError 的可选项。
type ErrorOption func(*Error)

// NewError 创建包含 HTTP 状态码和客户端说明的错误。
// 错误码默认由状态码决定，例如 404 是 not_found，400 是 bad_request。
func NewError(status int, message string, opts ...ErrorOption) *Error {
	err := &Error{
		status:  status,
		code:    defaultErrorCode(status),
		message: message,
	}

	if status < 400 || status > 599 {
		err.status = http.StatusInternalServerError
		err.message = "internal server error"
		err.code = defaultErrorCode(http.StatusInternalServerError)

		return err
	}

	for _, opt := range opts {
		if opt != nil {
			opt(err)
		}
	}
	return err
}

// WithCode 覆盖默认错误码。空字符串会被忽略。
func WithCode(code string) ErrorOption {
	return func(err *Error) {
		if err == nil || code == "" {
			return
		}
		err.code = code
	}
}

// WithDetails 设置返回给客户端的结构化详情。
func WithDetails(details any) ErrorOption {
	return func(err *Error) {
		if err == nil {
			return
		}
		err.details = details
	}
}

func (e *Error) Error() string {
	return e.message
}

func (e *Error) HTTPStatus() int { return e.status }

func (e *Error) ErrorCode() string {
	if e.code != "" {
		return e.code
	}
	return defaultErrorCode(e.status)
}

func (e *Error) ErrorMessage() string { return e.message }
func (e *Error) ErrorDetails() any    { return e.details }

func defaultErrorCode(status int) string {
	switch status {
	case http.StatusBadRequest:
		return "bad_request"
	case http.StatusUnauthorized:
		return "unauthorized"
	case http.StatusForbidden:
		return "forbidden"
	case http.StatusNotFound:
		return "not_found"
	case http.StatusMethodNotAllowed:
		return "method_not_allowed"
	case http.StatusConflict:
		return "conflict"
	case http.StatusRequestEntityTooLarge:
		return "request_too_large"
	case http.StatusUnsupportedMediaType:
		return "unsupported_media_type"
	case http.StatusUnprocessableEntity:
		return "unprocessable_entity"
	case http.StatusTooManyRequests:
		return "too_many_requests"
	case http.StatusInternalServerError:
		return "internal_error"
	default:
		return fmt.Sprintf("http_%d", status)
	}
}

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
