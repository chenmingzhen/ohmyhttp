package ohmyhttp

import "net/http"

// endpointExecutor 执行强类型端点的绑定、校验和中间件链。
type endpointExecutor struct {
	runtime      *appRuntime
	newRequest   func() any
	routeInfo    RouteInfo
	endpoint     EndpointNext
	maxBodyBytes int64
}

func newEndpointExecutor(
	runtime *appRuntime,
	route endpointRoute,
	appMiddleware []EndpointMiddleware,
) *endpointExecutor {
	middleware := make(
		[]EndpointMiddleware,
		0,
		len(appMiddleware)+len(route.endpointMiddleware),
	)
	middleware = append(middleware, appMiddleware...)
	middleware = append(middleware, route.endpointMiddleware...)
	maxBodyBytes := runtime.maxBodyBytes
	if route.maxBodyBytes != nil {
		maxBodyBytes = *route.maxBodyBytes
	}

	return &endpointExecutor{
		runtime:    runtime,
		newRequest: route.newRequest,
		routeInfo: RouteInfo{
			Kind:     RouteKindEndpoint,
			Method:   route.method,
			Pattern:  route.pattern,
			Metadata: route.metadata,
		},
		endpoint:     chainEndpoint(route.endpoint, middleware...),
		maxBodyBytes: maxBodyBytes,
	}
}

func (executor *endpointExecutor) execute(
	writer http.ResponseWriter,
	request *http.Request,
) routeResult {
	if executor.maxBodyBytes > 0 && request.Body != nil {
		request.Body = http.MaxBytesReader(writer, request.Body, executor.maxBodyBytes)
	}

	input := executor.newRequest()
	if err := executor.runtime.binder.Bind(request, input); err != nil {
		requestErr := &RequestBindingError{Cause: err}
		executor.runtime.observeRequest(RequestObservation{
			Request: request,
			Route:   executor.routeInfo,
			Input:   input,
			Err:     requestErr,
		})
		return routeResult{
			request: request,
			err:     requestErr,
		}
	}
	if err := executor.runtime.validator.Validate(request.Context(), input); err != nil {
		requestErr := normalizeValidationError(err)
		executor.runtime.observeRequest(RequestObservation{
			Request: request,
			Route:   executor.routeInfo,
			Input:   input,
			Err:     requestErr,
		})
		return routeResult{
			request: request,
			err:     requestErr,
		}
	}
	executor.runtime.observeRequest(RequestObservation{
		Request: request,
		Route:   executor.routeInfo,
		Input:   input,
	})

	ctx := &EndpointContext{
		Request: request,
		Route:   executor.routeInfo,
		input:   input,
	}
	response, err := executor.endpoint(ctx)
	if err != nil {
		return routeResult{request: ctx.Request, err: err}
	}
	return routeResult{
		request:  ctx.Request,
		response: response,
	}
}
