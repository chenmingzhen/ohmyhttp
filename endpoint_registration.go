package ohmyhttp

import (
	"context"
	"net/http"
	"reflect"
)

// Endpoint 是路由注册方法接受的强类型业务端点。
type Endpoint[Req, Resp any] func(context.Context, *Req) (Resp, error)

func newEndpointRoute[Req, Resp any](
	method string,
	pattern string,
	endpoint Endpoint[Req, Resp],
	options ...EndpointOption,
) endpointRoute {
	route := endpointRoute{
		method:        method,
		pattern:       pattern,
		successStatus: http.StatusOK,
		requestType:   reflect.TypeOf((*Req)(nil)),
		responseType:  reflect.TypeOf((*Resp)(nil)).Elem(),
		// 路由注册时只保存工厂；每次实际收到 HTTP 请求时，框架才调用它，
		// 得到一个互不共享的空请求对象 &Req{}。
		newRequest: func() any {
			return new(Req)
		},
	}
	for _, option := range options {
		option.applyEndpoint(&route)
	}
	successStatus := route.successStatus
	route.endpoint = func(ctx *EndpointContext) (Response, error) {
		typed := ctx.input.(*Req)
		result, err := endpoint(ctx.Context(), typed)
		if err != nil {
			return Response{}, err
		}
		return responseFromResult(result, successStatus), nil
	}
	return route
}
