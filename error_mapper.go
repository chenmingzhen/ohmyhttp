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
// 错误类型本身定义在 error.go。本文件不引用 errorx。
// 带 Unwrap 的调用栈包装会按内层错误继续匹配，不会因为外层有堆栈就改成 500。
type ErrorMapper interface {
	MapError(context.Context, error) Response
}

// defaultErrorMapper 是未调用 WithErrorMapper 时使用的映射器。
// 它只返回固定的 JSON Problem，不把内部错误文本写给客户端。
// 校验失败默认是 400；要使用 422，替换这个映射器。
type defaultErrorMapper struct{}

var _ ErrorMapper = defaultErrorMapper{}

// MapError 按下面的顺序匹配错误链，先匹配到的规则生效：
//
//   - HTTPError：使用它自己的状态码、错误码、说明和详情。状态码不在 400 到 599 时改为 500。
//   - *ValidationError：400 validation_failed，Details 原样放入 Problem.Details。
//   - *http.MaxBytesError：413 request_too_large。请求体超过 MaxBodyBytes 时由 net/http 产生。
//   - *UnsupportedMediaTypeError：415，Message 使用该错误的 Error()。
//   - *RequestBindingError：400 invalid_request。Cause 不返回给客户端。
//   - 其他错误，包括 *PanicError：500 internal_error。响应不包含 err.Error() 的文本。
func (defaultErrorMapper) MapError(_ context.Context, err error) Response {
	var httpError HTTPError
	if errors.As(err, &httpError) {
		status := httpError.HTTPStatus()
		if status < 400 || status > 599 {
			return JSON(http.StatusInternalServerError, Problem{
				Code:    "internal_error",
				Message: "internal server error",
			})
		}
		return JSON(status, Problem{
			Code:    httpError.ErrorCode(),
			Message: httpError.ErrorMessage(),
			Details: httpError.ErrorDetails(),
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

// mapError 调用当前 ErrorMapper，并保证结果是 4xx 或 5xx。
// 映射器返回的状态码不在 400 到 599 时，改成 500 internal_error。
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
