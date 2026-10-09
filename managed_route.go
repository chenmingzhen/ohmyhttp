package ohmyhttp

import (
	"fmt"
	"net/http"
)

// managedRoute 运行时的HTTP handler，统一接收 HTTP 请求，写入请求元数据，执行路由函数，并完成错误映射、响应转换、编码和 Observer 调用
type managedRoute struct {
	runtime *appRuntime
	route   RouteInfo
	run     routeFunc
}

type routeResult struct {
	request  *http.Request
	response Response
	err      error
}

type routeFunc func(http.ResponseWriter, *http.Request) routeResult

var _ http.Handler = (*managedRoute)(nil)

func newManagedRoute(
	runtime *appRuntime,
	route RouteInfo,
	run routeFunc,
) http.Handler {
	return &managedRoute{
		runtime: runtime,
		route:   route,
		run:     run,
	}
}

func adaptResponseFunc(responseFunc ResponseFunc) routeFunc {
	return func(_ http.ResponseWriter, request *http.Request) routeResult {
		response, err := responseFunc(request)
		return routeResult{request: request, response: response, err: err}
	}
}

func (route *managedRoute) ServeHTTP(
	writer http.ResponseWriter,
	request *http.Request,
) {
	request = withRequestMeta(request.Context(), request, route.route)
	defer func() {
		if request.MultipartForm != nil {
			_ = request.MultipartForm.RemoveAll()
		}
	}()

	result := route.run(writer, request)
	if result.request != nil {
		request = result.request
	}
	route.writeResponse(writer, request, result.response, result.err)
}

func (route *managedRoute) writeResponse(
	writer http.ResponseWriter,
	request *http.Request,
	response Response,
	requestErr error,
) {
	// writeResponse 负责把业务端点返回的
	//
	//     (Response, error)
	//
	// 转换成真正写入 http.ResponseWriter 的 HTTP 响应。
	// 整体流程是：
	//
	//   1. 业务错误通过 ErrorMapper 转成 Response；正常结果统一整理成 Response。
	//   2. 对可转换的 Endpoint 响应应用统一响应包装。
	//   3. 尝试编码并写出响应。
	//   4. 如果编码失败，根据响应是否已经提交，决定还能不能返回新的错误响应。
	ctx := request.Context()

	// 默认使用注册路由上的信息。如果请求上下文中有更新后的路由元数据，
	// 则使用上下文中的版本，确保 Observer 看到最终的路由信息。
	routeInfo := route.route
	if meta, ok := MetaFromContext(ctx); ok {
		routeInfo = meta.Route
	}

	// 业务函数返回 error 时，不直接把原始 error 写给客户端，而是交给
	// ErrorMapper 转成安全、稳定的 HTTP 错误响应，例如 404 或 500。
	// 正常返回时，则把普通返回值、Response 或 *Response 统一整理成 Response，
	// 默认成功状态码为 200。
	if requestErr != nil {
		response = route.runtime.mapError(ctx, requestErr)
	} else {
		response = responseFromResult(response, http.StatusOK)
	}

	// 对 Endpoint 的结构化响应应用统一包装，例如把业务数据包装成：
	//
	//     {"success": true, "data": ...}
	//
	// 文件、重定向、原始字节、流式响应以及不允许响应体的状态码不会进入
	// 这个转换流程，具体规则由 transformEndpointResponse 决定。
	response = route.runtime.transformEndpointResponse(
		ctx,
		routeInfo,
		response,
		requestErr,
	)

	// 业务错误已经完成映射和响应转换，现在通知错误观察器。
	// 观察器只负责记录错误或指标，不负责修改响应。
	if requestErr != nil {
		route.runtime.observeError(ctx, requestErr, response)
	}

	// Encoder 根据 Response.Body 的类型选择实际写法：普通对象编码为 JSON，
	// RawBody 写原始字节，FileBody 输出文件，RedirectBody 写重定向，
	// StreamingBody 执行流式写入。
	encodeErr := route.runtime.encoder.Encode(
		ctx,
		writer,
		request,
		response,
	)

	// 编码成功：响应已经正常完成，通知 ResponseObserver 后结束流程。
	if encodeErr == nil {
		route.runtime.observeResponse(
			ctx,
			routeInfo,
			response,
			requestErr,
		)
		return
	}

	// 编码失败时，先判断响应是否已经提交给客户端。
	// 例如流式响应可能已经写出了一部分内容，此时 HTTP 状态码和响应头
	// 已经发送，不能再追加一个新的 500 响应，否则会破坏 HTTP 响应格式。
	if responseCommitted(writer) {
		route.runtime.observeError(ctx, encodeErr, response)
		route.runtime.observeResponse(
			ctx,
			routeInfo,
			response,
			encodeErr,
		)
		return
	}

	// 到这里说明编码失败，但响应头还没有提交，客户端尚未收到响应。
	// 因此可以把编码错误包装成新的框架错误，并生成一个错误响应返回给客户端。
	finalErr := fmt.Errorf("encode response: %w", encodeErr)
	response = route.runtime.mapError(ctx, finalErr)
	response = route.runtime.transformEndpointResponse(
		ctx,
		routeInfo,
		response,
		finalErr,
	)

	// 记录最终的编码错误，然后尝试写出错误响应。错误响应本身如果再次编码失败，
	// encodeErrorResponse 只会记录该错误，不会继续递归生成新的错误响应。
	route.runtime.observeError(ctx, finalErr, response)
	route.runtime.encodeErrorResponse(
		ctx,
		writer,
		request,
		response,
	)

	// 无论正常响应还是编码失败后的错误响应，最终都会通知 ResponseObserver。
	route.runtime.observeResponse(
		ctx,
		routeInfo,
		response,
		finalErr,
	)
}
