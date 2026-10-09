package ohmyhttp

import (
	"context"
	"net/http"
)

// EndpointContext 是在业务中间件链中传递的上下文
type EndpointContext struct {
	Request *http.Request
	Route   RouteInfo

	input any
}

// EndpointNext 表示端点中间件链中的下一环。
type EndpointNext func(*EndpointContext) (Response, error)

// EndpointMiddleware 在请求解码和校验后执行。
type EndpointMiddleware func(*EndpointContext, EndpointNext) (Response, error)

// Context 返回当前 HTTP 请求携带的 context.Context。
func (ctx *EndpointContext) Context() context.Context {
	return ctx.Request.Context()
}

// SetContext 替换 HTTP Request 的上下文，并保留 ohmyhttp 请求元数据。
func (ctx *EndpointContext) SetContext(next context.Context) {
	ctx.Request = withRequestMeta(next, ctx.Request, ctx.Route)
}

// InputAs 以具体类型返回解码后的请求值。
func InputAs[T any](ctx *EndpointContext) (T, bool) {
	var zero T
	if ctx == nil {
		return zero, false
	}
	input, ok := ctx.input.(T)
	return input, ok
}

func chainEndpoint(
	endpoint EndpointNext,
	middleware ...EndpointMiddleware,
) EndpointNext {
	for i := len(middleware) - 1; i >= 0; i-- {
		current := middleware[i]
		next := endpoint
		endpoint = func(ctx *EndpointContext) (Response, error) {
			return current(ctx, next)
		}
	}
	return endpoint
}
