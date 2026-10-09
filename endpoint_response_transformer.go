package ohmyhttp

import (
	"context"
	"mime"
	"net/http"
	"reflect"
	"strings"
)

// EndpointResponseTransformer 在强类型 Endpoint 完成错误映射后、编码响应前转换最终响应。
// 文件、重定向、原始字节、流式响应和不允许响应体的状态码不会进入转换器。
type EndpointResponseTransformer interface {
	TransformEndpointResponse(context.Context, RouteInfo, Response, error) Response
}

// EndpointResponseTypeTransformer 可选地描述转换后的响应类型，供 OpenAPI 等工具
// 生成与实际响应一致的结构。
type EndpointResponseTypeTransformer interface {
	TransformEndpointResponseType(status int, bodyType reflect.Type) reflect.Type
}

// app := ohmyhttp.New(
//     ohmyhttp.WithEndpointResponseTransformer(
//         ohmyhttp.EndpointResponseTransformFunc(
//             func(
//                 ctx context.Context,
//                 route ohmyhttp.RouteInfo,
//                 response ohmyhttp.Response,
//                 err error,
//             ) ohmyhttp.Response {
//                 if err != nil {
//                     return response
//                 }

//                 response.Body = map[string]any{
//                     "success": true,
//                     "data":    response.Body,
//                 }
//                 return response
//             },
//         ),
//     ),
// )

// EndpointResponseTransformFunc 把函数适配为 EndpointResponseTransformer。
type EndpointResponseTransformFunc func(
	context.Context,
	RouteInfo,
	Response,
	error,
) Response

func (transform EndpointResponseTransformFunc) TransformEndpointResponse(
	ctx context.Context,
	route RouteInfo,
	response Response,
	err error,
) Response {
	return transform(ctx, route, response, err)
}

var _ EndpointResponseTransformer = EndpointResponseTransformFunc(nil)

var responseType = reflect.TypeOf(Response{})

func (runtime *appRuntime) transformEndpointInfo(info EndpointInfo) EndpointInfo {
	transformer, ok := runtime.endpointResponseTransformer.(EndpointResponseTypeTransformer)
	if !ok {
		return info
	}

	status := info.SuccessStatus
	if status == 0 {
		status = http.StatusOK
	}
	if isTransformableEndpointResponseType(status, info.ResponseType, jsonMediaType, false) {
		info.ResponseType = transformer.TransformEndpointResponseType(status, info.ResponseType)
	}
	for index := range info.Metadata.Returns {
		item := &info.Metadata.Returns[index]
		if isTransformableEndpointResponseType(
			item.Status,
			item.BodyType,
			item.ContentType,
			item.Binary,
		) {
			item.BodyType = transformer.TransformEndpointResponseType(item.Status, item.BodyType)
		}
	}
	return info
}

// isTransformableEndpointResponseType 判断是否可以转化为特定的结构化响应
func isTransformableEndpointResponseType(
	status int,
	bodyType reflect.Type,
	contentType string,
	binary bool,
) bool {
	return !binary &&
		bodyType != nil &&
		!statusDisallowsBody(status) &&
		isJSONResponseMediaType(contentType) &&
		dereferenceType(bodyType) != responseType
}

func isJSONResponseMediaType(value string) bool {
	if value == "" {
		return true
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return false
	}
	return mediaType == jsonMediaType || strings.HasSuffix(mediaType, "+json")
}

func dereferenceType(value reflect.Type) reflect.Type {
	for value != nil && value.Kind() == reflect.Pointer {
		value = value.Elem()
	}
	return value
}

func (runtime *appRuntime) transformEndpointResponse(
	ctx context.Context,
	route RouteInfo,
	response Response,
	err error,
) Response {
	if runtime.endpointResponseTransformer == nil ||
		route.Kind != RouteKindEndpoint ||
		!isTransformableEndpointResponse(response) {
		return response
	}
	return runtime.endpointResponseTransformer.TransformEndpointResponse(
		ctx,
		route,
		response,
		err,
	)
}

func isTransformableEndpointResponse(response Response) bool {
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}
	if statusDisallowsBody(status) {
		return false
	}
	switch response.Body.(type) {
	case FileBody, RedirectBody, RawBody, StreamingBody:
		return false
	default:
		return true
	}
}
