package ohmyhttp

import (
	"net/http"
	"runtime/debug"
	"strings"
)

// PanicMapper 把 recovery 捕获到的 panic 转换为响应。
type PanicMapper func(*http.Request, *PanicError) Response

// WithRecovery 启用 panic recovery。panic 会通过 ErrorMapper 生成响应。
func WithRecovery() AppOption {
	return func(runtime *appRuntime) {
		runtime.recoverPanics = true
	}
}

// WithPanicMapper 启用 panic recovery，并使用 mapper 生成响应。
func WithPanicMapper(mapper PanicMapper) AppOption {
	return func(runtime *appRuntime) {
		runtime.recoverPanics = true
		runtime.panicMapper = mapper
	}
}

func withRecovery(
	runtime *appRuntime,
	next http.Handler,
	panicMapper PanicMapper,
	routes map[string]RouteInfo,
) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		defer func() {
			recovered := recover()
			if recovered == nil {
				return
			}
			if recovered == http.ErrAbortHandler {
				panic(recovered)
			}

			route, ok := routes[request.Pattern]
			if !ok {
				pattern := request.Pattern
				pattern = strings.TrimPrefix(pattern, request.Method+" ")
				if pattern == "" {
					pattern = request.URL.Path
				}
				route = RouteInfo{
					Method:  request.Method,
					Pattern: pattern,
				}
			}
			if route.Method == "" {
				route.Method = request.Method
			}
			if route.Pattern == "" {
				route.Pattern = strings.TrimPrefix(
					request.Pattern,
					request.Method+" ",
				)
				if route.Pattern == "" {
					route.Pattern = request.URL.Path
				}
			}
			request = withRequestMeta(request.Context(), request, route)
			panicErr := &PanicError{Value: recovered, Stack: debug.Stack()}
			var response Response
			if panicMapper == nil {
				response = runtime.mapError(request.Context(), panicErr)
			} else {
				response = panicMapper(request, panicErr)
				if response.Status == 0 {
					response.Status = http.StatusInternalServerError
				}
			}
			response = runtime.transformEndpointResponse(
				request.Context(),
				route,
				response,
				panicErr,
			)
			runtime.observeError(request.Context(), panicErr, response)
			if !responseCommitted(writer) {
				runtime.encodeErrorResponse(
					request.Context(),
					writer,
					request,
					response,
				)
			}
			runtime.observeResponse(
				request.Context(),
				route,
				response,
				panicErr,
			)
		}()

		next.ServeHTTP(writer, request)
	})
}
