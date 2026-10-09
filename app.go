package ohmyhttp

import "net/http"

// App 是直接构建在 net/http 上的强类型 HTTP 应用
type App struct {
	runtime *appRuntime

	endpointRoutes []endpointRoute // 注册来源:Get、Post 等
	responseRoutes []responseRoute // 注册来源:HandleResponse、Static
	httpRoutes     []httpRoute     // 注册来源:HandleHTTP

	middleware         []Middleware
	endpointMiddleware []EndpointMiddleware

	httpHandler http.Handler
}

var _ http.Handler = (*App)(nil)

func New(options ...AppOption) *App {
	return &App{runtime: newAppRuntime(options...)}
}

// Use 添加覆盖全部 HTTP 请求的标准 HTTP 中间件。
func (app *App) Use(middleware ...Middleware) {
	app.middleware = append(app.middleware, middleware...)
}

// UseEndpointMiddleware 添加绑定和校验后执行的端点中间件。
func (app *App) UseEndpointMiddleware(middleware ...EndpointMiddleware) {
	app.endpointMiddleware = append(app.endpointMiddleware, middleware...)
}

// Endpoints 返回已注册强类型端点的描述快照。响应类型已经应用配置的响应类型转换器。
func (app *App) Endpoints() []EndpointInfo {
	endpoints := make([]EndpointInfo, len(app.endpointRoutes))
	for index, route := range app.endpointRoutes {
		endpoints[index] = app.runtime.transformEndpointInfo(route.info())
	}
	return endpoints
}

func (app *App) Environment() Environment {
	return app.runtime.environment
}

// HandleResponse 注册不进行结构体绑定、但由框架统一编码 Response 和 error 的路由。
func (app *App) HandleResponse(
	method string,
	pattern string,
	responseFunc ResponseFunc,
	options ...ResponseRouteOption,
) {
	route := newResponseRoute(method, pattern, responseFunc, options...)
	app.registerResponseRoute(route)
}

// ServeHTTP 实现 http.Handler。
func (app *App) ServeHTTP(writer http.ResponseWriter, request *http.Request) {
	if app.httpHandler == nil {
		app.httpHandler = app.createHTTPHandler()
	}
	tracked := &trackedResponseWriter{ResponseWriter: writer}
	app.httpHandler.ServeHTTP(tracked, request)
}

func (app *App) registerEndpointRoute(route endpointRoute) {
	app.endpointRoutes = append(app.endpointRoutes, route)
}

func (app *App) registerResponseRoute(route responseRoute) {
	app.responseRoutes = append(app.responseRoutes, route)
}

func (app *App) registerHTTPRoute(route httpRoute) {
	app.httpRoutes = append(app.httpRoutes, route)
}
