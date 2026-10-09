package ohmyhttp

import "net/http"

// Handle 在 App 上注册使用任意 HTTP 方法的强类型路由。
func (app *App) Handle[Req, Resp any](
	method string,
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(method, pattern, endpoint, options...)
	app.registerEndpointRoute(route)
}

// Get 在 App 上注册 GET 路由。
func (app *App) Get[Req, Resp any](
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(http.MethodGet, pattern, endpoint, options...)
	app.registerEndpointRoute(route)
}

// Post 在 App 上注册 POST 路由。
func (app *App) Post[Req, Resp any](
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(http.MethodPost, pattern, endpoint, options...)
	app.registerEndpointRoute(route)
}

// Put 在 App 上注册 PUT 路由。
func (app *App) Put[Req, Resp any](
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(http.MethodPut, pattern, endpoint, options...)
	app.registerEndpointRoute(route)
}

// Patch 在 App 上注册 PATCH 路由。
func (app *App) Patch[Req, Resp any](
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(http.MethodPatch, pattern, endpoint, options...)
	app.registerEndpointRoute(route)
}

// Delete 在 App 上注册 DELETE 路由。
func (app *App) Delete[Req, Resp any](
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(http.MethodDelete, pattern, endpoint, options...)
	app.registerEndpointRoute(route)
}

// Handle 在 Group 上注册使用任意 HTTP 方法的强类型路由。
func (group *Group) Handle[Req, Resp any](
	method string,
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(method, pattern, endpoint, options...)
	group.registerEndpointRoute(route)
}

// Get 在 Group 上注册 GET 路由。
func (group *Group) Get[Req, Resp any](
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(http.MethodGet, pattern, endpoint, options...)
	group.registerEndpointRoute(route)
}

// Post 在 Group 上注册 POST 路由。
func (group *Group) Post[Req, Resp any](
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(http.MethodPost, pattern, endpoint, options...)
	group.registerEndpointRoute(route)
}

// Put 在 Group 上注册 PUT 路由。
func (group *Group) Put[Req, Resp any](
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(http.MethodPut, pattern, endpoint, options...)
	group.registerEndpointRoute(route)
}

// Patch 在 Group 上注册 PATCH 路由。
func (group *Group) Patch[Req, Resp any](
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(http.MethodPatch, pattern, endpoint, options...)
	group.registerEndpointRoute(route)
}

// Delete 在 Group 上注册 DELETE 路由。
func (group *Group) Delete[Req, Resp any](
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) {
	route := newEndpointRoute(http.MethodDelete, pattern, endpoint, options...)
	group.registerEndpointRoute(route)
}
