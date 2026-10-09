package ohmyhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/chenmingzhen/ohmyhttp/errorx"
)

// Problem 是默认的 JSON 错误结构。
type Problem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}

// PublicError 由可以安全暴露给客户端的错误实现。
type PublicError interface {
	error
	HTTPStatus() int
	ErrorCode() string
	PublicMessage() string
	ErrorDetails() any
}

// Error 是默认的公开应用错误实现。
type Error struct {
	status  int
	code    string
	message string
	details any
}

var _ PublicError = (*Error)(nil)

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

func (e *Error) HTTPStatus() int       { return e.status }
func (e *Error) ErrorCode() string     { return e.code }
func (e *Error) PublicMessage() string { return e.message }
func (e *Error) ErrorDetails() any     { return e.details }

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

var (
	_ error              = (*PanicError)(nil)
	_ errorx.StackTracer = (*PanicError)(nil)
)

func (e *PanicError) Error() string { return fmt.Sprintf("panic recovered: %v", e.Value) }

func (e *PanicError) StackTrace() string { return string(e.Stack) }

// ErrorMapper 把错误转换为框架无关的 HTTP 响应。
type ErrorMapper interface {
	MapError(context.Context, error) Response
}

// defaultErrorMapper 返回安全、稳定的 JSON 错误。校验错误默认使用 HTTP 400；如果
// 应用希望使用 422，可以替换错误映射器。
type defaultErrorMapper struct{}

var _ ErrorMapper = defaultErrorMapper{}

func (defaultErrorMapper) MapError(_ context.Context, err error) Response {
	var stackErr *errorx.StackError
	if errors.As(err, &stackErr) {
		return JSON(http.StatusInternalServerError, Problem{
			Code:    "internal_error",
			Message: "internal server error",
		})
	}

	var public PublicError
	if errors.As(err, &public) {
		status := public.HTTPStatus()
		if status < 400 || status > 599 {
			return JSON(http.StatusInternalServerError, Problem{
				Code:    "internal_error",
				Message: "internal server error",
			})
		}
		return JSON(status, Problem{
			Code:    public.ErrorCode(),
			Message: public.PublicMessage(),
			Details: public.ErrorDetails(),
		})
	}

	var validation *ValidationError
	if errors.As(err, &validation) {
		return JSON(http.StatusBadRequest, Problem{
			Code:    "validation_failed",
			Message: "request validation failed",
			Details: validation.Details,
		})
	}

	var maxBytes *http.MaxBytesError
	if errors.As(err, &maxBytes) {
		return JSON(http.StatusRequestEntityTooLarge, Problem{
			Code:    "request_too_large",
			Message: "request body is too large",
		})
	}

	var unsupported *UnsupportedMediaTypeError
	if errors.As(err, &unsupported) {
		return JSON(http.StatusUnsupportedMediaType, Problem{
			Code:    "unsupported_media_type",
			Message: unsupported.Error(),
		})
	}

	var binding *RequestBindingError
	if errors.As(err, &binding) {
		return JSON(http.StatusBadRequest, Problem{
			Code:    "invalid_request",
			Message: "request could not be decoded",
		})
	}

	return JSON(http.StatusInternalServerError, Problem{
		Code:    "internal_error",
		Message: "internal server error",
	})
}

func (runtime *appRuntime) mapError(ctx context.Context, err error) Response {
	response := runtime.errorMapper.MapError(ctx, err)
	if response.Status >= 400 && response.Status <= 599 {
		return response
	}
	return JSON(http.StatusInternalServerError, Problem{
		Code:    "internal_error",
		Message: "internal server error",
	})
}

func (runtime *appRuntime) encodeErrorResponse(
	ctx context.Context,
	writer http.ResponseWriter,
	request *http.Request,
	response Response,
) {
	if err := runtime.encoder.Encode(ctx, writer, request, response); err != nil {
		runtime.observeError(
			ctx,
			fmt.Errorf("encode error response: %w", err),
			response,
		)
	}
}
