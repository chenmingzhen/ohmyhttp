package ohmyhttp

import (
	"net/http"
	"slices"
	"strings"
)

// MethodNotAllowedFunc 为路径存在但方法不匹配的请求生成响应。
type MethodNotAllowedFunc func(*http.Request, []string) (Response, error)

const fallbackRoutePattern = "/"

func fallbackResponseFunc(
	runtime *appRuntime,
	mux *http.ServeMux,
	routes map[string]RouteInfo,
) ResponseFunc {
	methods := collectRouteMethods(routes)

	return func(request *http.Request) (Response, error) {
		allowed := matchingMethods(mux, request, methods)
		if len(allowed) > 0 {
			return methodNotAllowedResponse(runtime, request, allowed)
		}
		return notFoundResponse(runtime, request)
	}
}

func collectRouteMethods(routes map[string]RouteInfo) []string {
	methodSet := make(map[string]struct{})
	for _, route := range routes {
		if route.Method == "" {
			continue
		}
		methodSet[route.Method] = struct{}{}
		if route.Method == http.MethodGet {
			methodSet[http.MethodHead] = struct{}{}
		}
	}

	methods := make([]string, 0, len(methodSet))
	for method := range methodSet {
		methods = append(methods, method)
	}
	slices.Sort(methods)
	return methods
}

func matchingMethods(
	mux *http.ServeMux,
	request *http.Request,
	methods []string,
) []string {
	probe := request.Clone(request.Context())
	allowed := make([]string, 0)
	for _, method := range methods {
		if method == request.Method {
			continue
		}

		// 改掉 Method 交给 ServeMux 再匹配一次
		probe.Method = method
		_, pattern := mux.Handler(probe)
		// 如果这个路径能匹配到真实路由 就说明这个方法允许
		if pattern != "" && pattern != fallbackRoutePattern {
			allowed = append(allowed, method)
		}
	}
	return allowed
}

func methodNotAllowedResponse(
	runtime *appRuntime,
	request *http.Request,
	allowed []string,
) (Response, error) {
	allow := strings.Join(allowed, ", ")
	if custom := runtime.methodNotAllowed; custom != nil {
		response, err := custom(request, allowed)
		if err == nil && response.Header.Get("Allow") == "" {
			response = response.WithHeader("Allow", allow)
		}
		return response, err
	}
	return JSON(http.StatusMethodNotAllowed, Problem{
		Code:    "method_not_allowed",
		Message: "method not allowed",
	}).WithHeader("Allow", allow), nil
}

func notFoundResponse(
	runtime *appRuntime,
	request *http.Request,
) (Response, error) {
	if custom := runtime.notFound; custom != nil {
		return custom(request)
	}
	return JSON(http.StatusNotFound, Problem{
		Code:    "not_found",
		Message: "route not found",
	}), nil
}
