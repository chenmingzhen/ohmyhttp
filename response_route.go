package ohmyhttp

import "net/http"

// ResponseFunc 处理已经匹配到的 HTTP 请求。它可以读取原始请求，但不能直接写
// ResponseWriter；所有正常响应和错误都由 App 统一编码。
type ResponseFunc func(*http.Request) (Response, error)

// ResponseRouteOption 配置不进行请求绑定的 ResponseFunc 路由。
type ResponseRouteOption interface {
	applyResponseRoute(*responseRoute)
}

type responseRoute struct {
	method   string
	pattern  string
	kind     RouteKind
	metadata RouteMetadata
	respond  ResponseFunc
}

func newResponseRoute(
	method string,
	pattern string,
	responseFunc ResponseFunc,
	options ...ResponseRouteOption,
) responseRoute {
	route := responseRoute{
		method:  method,
		pattern: pattern,
		kind:    RouteKindResponse,
		respond: responseFunc,
	}
	for _, option := range options {
		option.applyResponseRoute(&route)
	}
	return route
}
