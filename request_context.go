package ohmyhttp

import (
	"context"
	"net/http"
)

// RequestMeta 是框架无关的请求元数据，通过 context.Context 提供给端点中间件和
// 业务 Endpoint。
type RequestMeta struct {
	Request *http.Request
	Route   RouteInfo
}

// MetaFromContext 返回当前 HTTP 请求的元数据。
func MetaFromContext(ctx context.Context) (RequestMeta, bool) {
	if ctx == nil {
		return RequestMeta{}, false
	}
	meta, ok := ctx.Value(requestMetaKey{}).(RequestMeta)
	if !ok {
		return RequestMeta{}, false
	}
	return meta, true
}

// PathParam 从请求元数据中读取路径参数。
func PathParam(ctx context.Context, name string) string {
	meta, ok := MetaFromContext(ctx)
	if !ok || meta.Request == nil {
		return ""
	}
	return meta.Request.PathValue(name)
}

type requestMetaKey struct{}

func withRequestMeta(
	ctx context.Context,
	request *http.Request,
	route RouteInfo,
) *http.Request {
	if route.Method == "" {
		route.Method = request.Method
	}
	if route.Pattern == "" {
		route.Pattern = request.Pattern
	}
	meta := RequestMeta{Request: request, Route: route}
	return request.WithContext(context.WithValue(ctx, requestMetaKey{}, meta))
}

func withRouteInfo(route RouteInfo, next http.Handler) http.Handler {
	return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		request = withRequestMeta(request.Context(), request, route)
		next.ServeHTTP(writer, request)
	})
}
