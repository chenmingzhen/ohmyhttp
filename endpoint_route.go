package ohmyhttp

import "reflect"

// EndpointInfo 是已注册强类型端点的描述快照，可用于生成文档或检查端点。
// ResponseType 和 Metadata.Returns 中的响应体类型均为最终线上响应类型。
type EndpointInfo struct {
	Method        string
	Pattern       string
	Metadata      RouteMetadata
	SuccessStatus int
	RequestType   reflect.Type
	ResponseType  reflect.Type
}

// EndpointOption 在注册阶段定制强类型端点。
type EndpointOption interface {
	applyEndpoint(*endpointRoute)
}

// Status 修改普通成功响应使用的 HTTP 状态码。
func Status(status int) EndpointOption {
	return endpointOptionFunc(func(route *endpointRoute) {
		route.successStatus = status
	})
}

// MaxBodyBytes 为单个路由设置请求体大小限制；非正数表示该路由不限制。
func MaxBodyBytes(limit int64) EndpointOption {
	return endpointOptionFunc(func(route *endpointRoute) {
		route.maxBodyBytes = &limit
	})
}

// WithEndpointMiddleware 为单个路由添加端点中间件。中间件按声明顺序在 App 和 Group
// 中间件之后、强类型 Endpoint 之前执行。
func WithEndpointMiddleware(middleware ...EndpointMiddleware) EndpointOption {
	return endpointOptionFunc(func(route *endpointRoute) {
		route.endpointMiddleware = append(route.endpointMiddleware, middleware...)
	})
}

type endpointRoute struct {
	method             string
	pattern            string
	metadata           RouteMetadata
	successStatus      int
	requestType        reflect.Type
	responseType       reflect.Type
	newRequest         func() any
	endpoint           EndpointNext
	endpointMiddleware []EndpointMiddleware
	maxBodyBytes       *int64
}

type endpointOptionFunc func(*endpointRoute)

var _ EndpointOption = endpointOptionFunc(nil)

func (route endpointRoute) info() EndpointInfo {
	return EndpointInfo{
		Method:        route.method,
		Pattern:       route.pattern,
		Metadata:      cloneRouteMetadata(route.metadata),
		SuccessStatus: route.successStatus,
		RequestType:   route.requestType,
		ResponseType:  route.responseType,
	}
}

func (option endpointOptionFunc) applyEndpoint(route *endpointRoute) { option(route) }
