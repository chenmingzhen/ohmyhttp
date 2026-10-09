package ohmyhttp

import "net/http"

// HandleHTTP 注册直接控制 ResponseWriter 的标准库 Handler。
func (app *App) HandleHTTP(method string, pattern string, handler http.Handler) {
	app.registerHTTPRoute(httpRoute{
		method:  method,
		pattern: pattern,
		handler: handler,
	})
}

// HandleHTTP 在当前分组下注册标准库 Handler。
func (group *Group) HandleHTTP(method string, pattern string, handler http.Handler) {
	hierarchy := group.configurationHierarchy()
	for _, item := range hierarchy {
		pattern = joinPath(item.prefix, pattern)
	}
	group.app.registerHTTPRoute(httpRoute{
		method:  method,
		pattern: pattern,
		handler: handler,
	})
}

type httpRoute struct {
	method  string
	pattern string
	handler http.Handler
}
