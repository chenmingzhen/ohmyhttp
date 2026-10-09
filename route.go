package ohmyhttp

// RouteKind 标识一条请求由哪一种执行管线处理。
type RouteKind string

const (
	RouteKindEndpoint RouteKind = "endpoint"
	RouteKindResponse RouteKind = "response"
	RouteKindStatic   RouteKind = "static"
	RouteKindHTTP     RouteKind = "http"
	RouteKindFallback RouteKind = "fallback"
)

// RouteInfo 标识匹配到的路由。
type RouteInfo struct {
	Kind     RouteKind
	Method   string
	Pattern  string
	Metadata RouteMetadata
}
