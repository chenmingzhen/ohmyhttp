package ohmyhttp

import "strings"

// Group 是注册期的路由分组工具，只负责添加路径前缀和端点中间件。
type Group struct {
	app                *App
	parent             *Group
	prefix             string
	endpointMiddleware []EndpointMiddleware
}

// Group 创建一个路由分组。prefix 为空或 "/" 时表示不添加路径前缀。
func (app *App) Group(prefix string) *Group {
	return newGroup(app, nil, prefix)
}

// Group 创建子分组。子分组会继承父分组的路径前缀和中间件。
func (group *Group) Group(prefix string) *Group {
	return newGroup(group.app, group, prefix)
}

// UseEndpointMiddleware 为当前分组后续注册的路由追加端点中间件。
func (group *Group) UseEndpointMiddleware(middleware ...EndpointMiddleware) {
	group.endpointMiddleware = append(group.endpointMiddleware, middleware...)
}

// HandleResponse 在当前分组下注册 ResponseFunc 路由。
func (group *Group) HandleResponse(
	method string,
	pattern string,
	responseFunc ResponseFunc,
	options ...ResponseRouteOption,
) {
	route := newResponseRoute(method, pattern, responseFunc, options...)
	group.registerResponseRoute(route)
}

func newGroup(
	app *App,
	parent *Group,
	prefix string,
) *Group {
	return &Group{
		app:    app,
		parent: parent,
		prefix: normalizeGroupPrefix(prefix),
	}
}

// registerEndpointRoute 一次性解析完整父链，把最终前缀和中间件应用到端点路由后
// 直接交给 App。
func (group *Group) registerEndpointRoute(route endpointRoute) {
	hierarchy := group.configurationHierarchy()

	for _, item := range hierarchy {
		route.pattern = joinPath(item.prefix, route.pattern)
	}

	totalEndpointMiddleware := len(route.endpointMiddleware)
	for _, item := range hierarchy {
		totalEndpointMiddleware += len(item.endpointMiddleware)
	}
	endpointMiddleware := make([]EndpointMiddleware, 0, totalEndpointMiddleware)
	for index := len(hierarchy) - 1; index >= 0; index-- {
		endpointMiddleware = append(
			endpointMiddleware,
			hierarchy[index].endpointMiddleware...,
		)
	}
	route.endpointMiddleware = append(
		endpointMiddleware,
		route.endpointMiddleware...,
	)
	group.app.registerEndpointRoute(route)
}

func (group *Group) registerResponseRoute(route responseRoute) {
	hierarchy := group.configurationHierarchy()
	for _, item := range hierarchy {
		route.pattern = joinPath(item.prefix, route.pattern)
	}
	group.app.registerResponseRoute(route)
}

// configurationHierarchy 返回 child -> root 顺序。这个顺序可直接用于
// 逐层拼接路径；端点中间件则按相反方向组合为 root -> child。
func (group *Group) configurationHierarchy() []*Group {
	var hierarchy []*Group
	for current := group; current != nil; current = current.parent {
		hierarchy = append(hierarchy, current)
	}
	return hierarchy
}

func normalizeGroupPrefix(prefix string) string {
	prefix = strings.TrimSpace(prefix)
	if prefix == "" || prefix == "/" {
		return ""
	}
	return strings.TrimRight(prefix, "/")
}

func joinPath(prefix string, pattern string) string {
	if prefix == "" {
		return pattern
	}
	if pattern == "/" {
		return prefix
	}
	return prefix + pattern
}
