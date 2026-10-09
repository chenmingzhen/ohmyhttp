package ohmyhttp

import (
	"context"
	"net/http"
)

// RequestObservation 是强类型端点完成请求绑定和校验后的只读快照。
// Input 和 Request 仍由请求链路持有，观察器不应修改或异步保留它们。
type RequestObservation struct {
	Request *http.Request
	Route   RouteInfo
	Input   any
	Err     error
}

// RequestObserver 在业务端点执行前接收绑定后的请求。绑定或校验失败时也会调用，
// 此时 Input 是部分绑定的请求对象，Err 是即将进入错误映射的错误。
type RequestObserver func(context.Context, RequestObservation)

// ErrorObserver 接收经过错误映射和 Endpoint 响应转换后的错误，可在不修改响应的情况下
// 记录堆栈或指标。
type ErrorObserver func(context.Context, error, Response)

// ResponseObserver 在托管响应完成编码后接收路由、最终响应和关联错误。
// 标准库 Handler 的正常响应不经过 Response 管线；由 Recovery 生成的 panic
// 响应仍会触发该观察器。
type ResponseObserver func(context.Context, RouteInfo, Response, error)

// ResponseInfo 是当前请求响应写入状态的只读快照。
type ResponseInfo struct {
	StatusCode   int
	BytesWritten int64
	Committed    bool
}

// ResponseInfoFromWriter 返回 App 传入 Middleware 的响应写入状态。
func ResponseInfoFromWriter(writer http.ResponseWriter) (ResponseInfo, bool) {
	for writer != nil {
		if tracker, ok := writer.(responseTracker); ok {
			status := tracker.StatusCode()
			if status == 0 {
				status = http.StatusOK
			}
			return ResponseInfo{
				StatusCode:   status,
				BytesWritten: tracker.BytesWritten(),
				Committed:    tracker.Written(),
			}, true
		}
		unwrapper, ok := writer.(interface {
			Unwrap() http.ResponseWriter
		})
		if !ok {
			break
		}
		next := unwrapper.Unwrap()
		if next == writer {
			break
		}
		writer = next
	}
	return ResponseInfo{}, false
}

type responseTracker interface {
	StatusCode() int
	BytesWritten() int64
	Written() bool
}

func responseCommitted(writer http.ResponseWriter) bool {
	response, ok := ResponseInfoFromWriter(writer)
	return ok && response.Committed
}

func (runtime *appRuntime) observeRequest(observation RequestObservation) {
	if runtime.requestObserver != nil {
		runtime.requestObserver(observation.Request.Context(), observation)
	}
}

func (runtime *appRuntime) observeError(
	ctx context.Context,
	err error,
	response Response,
) {
	if runtime.errorObserver != nil {
		runtime.errorObserver(ctx, err, response)
	}
}

func (runtime *appRuntime) observeResponse(
	ctx context.Context,
	route RouteInfo,
	response Response,
	err error,
) {
	if runtime.responseObserver != nil {
		runtime.responseObserver(ctx, route, response, err)
	}
}
