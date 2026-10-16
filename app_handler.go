package ohmyhttp

import "net/http"

func (app *App) createHTTPHandler() http.Handler {
	mux := http.NewServeMux()
	routes := map[string]RouteInfo{}

	// 装载业务路由
	for _, route := range app.endpointRoutes {
		executor := newEndpointExecutor(
			app.runtime,
			route,
			app.endpointMiddleware,
		)
		handler := newManagedRoute(
			app.runtime,
			executor.routeInfo,
			executor.execute,
		)
		mux.Handle(route.method+" "+route.pattern, handler)
		routes[route.method+" "+route.pattern] = executor.routeInfo
	}

	// 装载托管 Response 路由
	for _, route := range app.responseRoutes {
		routeInfo := RouteInfo{
			Kind:     route.kind,
			Method:   route.method,
			Pattern:  route.pattern,
			Metadata: route.metadata,
		}
		handler := newManagedRoute(
			app.runtime,
			routeInfo,
			adaptResponseFunc(route.respond),
		)
		mux.Handle(route.method+" "+route.pattern, handler)
		routes[route.method+" "+route.pattern] = routeInfo
	}

	// 装载标准库 Handler 路由
	for _, route := range app.httpRoutes {
		routeInfo := RouteInfo{
			Kind:    RouteKindHTTP,
			Method:  route.method,
			Pattern: route.pattern,
		}
		handler := withRouteInfo(routeInfo, route.handler)
		mux.Handle(route.method+" "+route.pattern, handler)
		routes[route.method+" "+route.pattern] = routeInfo
	}

	// 404 405 兜底
	fallbackRouteInfo := RouteInfo{Kind: RouteKindFallback}
	fallback := newManagedRoute(
		app.runtime,
		fallbackRouteInfo,
		adaptResponseFunc(fallbackResponseFunc(app.runtime, mux, routes)),
	)
	mux.Handle(fallbackRoutePattern, fallback)
	routes[fallbackRoutePattern] = fallbackRouteInfo

	var handler http.Handler = mux
	if app.runtime.recoverPanics {
		handler = withRecovery(
			app.runtime,
			handler,
			app.runtime.panicMapper,
			routes,
		)
	}
	return chainMiddleware(handler, app.middleware...)
}
