package ohmyhttp

import "net/http"

// Middleware 包装标准库 http.Handler。
type Middleware func(http.Handler) http.Handler

// ChainMiddleware 按声明顺序包装 Handler。第一个中间件位于最外层，最先接收请求、最后接收响应。
func ChainMiddleware(handler http.Handler, middleware ...Middleware) http.Handler {
	for index := len(middleware) - 1; index >= 0; index-- {
		handler = middleware[index](handler)
	}
	return handler
}
