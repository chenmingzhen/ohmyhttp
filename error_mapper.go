package ohmyhttp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

// ErrorMapper 把 error 转换成 Response。
//
// 用 WithErrorMapper 替换默认实现。这里返回的状态码必须在 400 到 599 之间；
// 超出该范围时，appRuntime.mapError 会改成 500 internal_error。
// 更换 JSON 形状时，在自定义映射里调用 DefaultMapError。错误类型定义在 error.go。
// 带 Unwrap 的调用栈包装会按内层错误继续匹配，不会因为外层有堆栈就改成 500。
type ErrorMapper interface {
	MapError(context.Context, error) Response
}

// ErrorInfo 是错误分类的结果，还没有写成响应正文。
type ErrorInfo struct {
	Status  int
	Code    string
	Message string
	Details any
}

// DefaultMapError 按错误类型生成响应。bodyMapper 把 ErrorInfo 写成 JSON 正文，决定响应里写出哪些字段。
// 状态码不在 400 到 599 时改为 500 internal_error。
//
// 匹配顺序：
//
//   - HTTPError：使用它自己的状态码、错误码、说明和详情。
//   - *ValidationError：400 validation_failed，Details 原样放入 ErrorInfo.Details。
//   - *http.MaxBytesError：413 request_too_large。
//   - *UnsupportedMediaTypeError：415，Message 使用该错误的 Error()。
//   - *RequestBindingError：400 invalid_request。Cause 不进入 ErrorInfo。
//   - 其他错误，包括 *PanicError：500 internal_error。Message 固定，不包含 err.Error()。
func DefaultMapError(err error, bodyMapper func(ErrorInfo) any) Response {
	info := describeError(err)

	if bodyMapper == nil {
		bodyMapper = problemBody
	}

	return JSON(info.Status, bodyMapper(info))
}

func problemBody(info ErrorInfo) any {
	return Problem{
		Code:    info.Code,
		Message: info.Message,
		Details: info.Details,
	}
}

func describeError(err error) ErrorInfo {
	var httpError HTTPError
	if errors.As(err, &httpError) {
		return ErrorInfo{
			Status:  httpError.HTTPStatus(),
			Code:    httpError.ErrorCode(),
			Message: httpError.ErrorMessage(),
			Details: httpError.ErrorDetails(),
		}
	}

	var validation *ValidationError
	if errors.As(err, &validation) {
		return ErrorInfo{
			Status:  http.StatusBadRequest,
			Code:    "validation_failed",
			Message: "request validation failed",
			Details: validation.Details,
		}
	}

	var maxBytes *http.MaxBytesError
	if errors.As(err, &maxBytes) {
		return ErrorInfo{
			Status:  http.StatusRequestEntityTooLarge,
			Code:    "request_too_large",
			Message: "request body is too large",
		}
	}

	var unsupported *UnsupportedMediaTypeError
	if errors.As(err, &unsupported) {
		return ErrorInfo{
			Status:  http.StatusUnsupportedMediaType,
			Code:    "unsupported_media_type",
			Message: unsupported.Error(),
		}
	}

	var binding *RequestBindingError
	if errors.As(err, &binding) {
		return ErrorInfo{
			Status:  http.StatusBadRequest,
			Code:    "invalid_request",
			Message: "request could not be decoded",
		}
	}

	return ErrorInfo{
		Status:  http.StatusInternalServerError,
		Code:    "internal_error",
		Message: "internal server error",
	}
}

// mapError 调用当前 ErrorMapper。未设置时用 DefaultMapError 写成 Problem。
// 映射结果的状态码不在 400 到 599 时，改成 500 internal_error。
func (runtime *appRuntime) mapError(ctx context.Context, err error) Response {
	if runtime.errorMapper == nil {
		return DefaultMapError(err, problemBody)
	}
	response := runtime.errorMapper.MapError(ctx, err)
	if response.Status >= 400 && response.Status <= 599 {
		return response
	}
	return JSON(http.StatusInternalServerError, Problem{
		Code:    "internal_error",
		Message: "internal server error",
	})
}

// encodeErrorResponse 写出已经映射好的错误响应。
// 写出再次失败时只通知错误观察器，不再生成新的错误响应，避免递归。
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
