package ohmyhttp

import (
	"context"

	"google.golang.org/protobuf/proto"
)

// AppOption 配置 App 的请求处理能力。
type AppOption func(*appRuntime)

// WithErrorMapper 将错误转换为框架的 Response
func WithErrorMapper(mapper ErrorMapper) AppOption {
	return func(runtime *appRuntime) {
		runtime.errorMapper = mapper
	}
}

// WithProtoValidator 设置 protobuf 请求的可选校验器。普通 struct 仍然
// 使用 validator/v10；实现 proto.Message 的请求才会进入这个校验器。
func WithProtoValidator(validator func(context.Context, proto.Message) error) AppOption {
	return func(runtime *appRuntime) {
		runtime.validator.protoValidator = validator
	}
}

// WithEndpointResponseTransformer 设置强类型 Endpoint 的最终响应转换器。
func WithEndpointResponseTransformer(transformer EndpointResponseTransformer) AppOption {
	return func(runtime *appRuntime) {
		runtime.endpointResponseTransformer = transformer
	}
}

// WithRequestObserver 观察强类型端点绑定和校验后的请求参数。
func WithRequestObserver(observer RequestObserver) AppOption {
	return func(runtime *appRuntime) {
		runtime.requestObserver = observer
	}
}

func WithErrorObserver(observer ErrorObserver) AppOption {
	return func(runtime *appRuntime) {
		runtime.errorObserver = observer
	}
}

// WithResponseObserver 观察所有托管路由的最终 Response。
func WithResponseObserver(observer ResponseObserver) AppOption {
	return func(runtime *appRuntime) {
		runtime.responseObserver = observer
	}
}

// WithNotFound 替换默认的 JSON 404 响应函数。
func WithNotFound(responseFunc ResponseFunc) AppOption {
	return func(runtime *appRuntime) {
		runtime.notFound = responseFunc
	}
}

// WithMethodNotAllowed 替换默认的 JSON 405 响应函数。
func WithMethodNotAllowed(responseFunc MethodNotAllowedFunc) AppOption {
	return func(runtime *appRuntime) {
		runtime.methodNotAllowed = responseFunc
	}
}

// WithDiscardUnknownFields 控制 JSON 请求体是否忽略未知字段。
func WithDiscardUnknownFields(discard bool) AppOption {
	return func(runtime *appRuntime) {
		runtime.binder.discardUnknown = discard
	}
}

// WithMultipartMemory 设置 multipart 表单解析时保留在内存中的最大字节数。
func WithMultipartMemory(memory int64) AppOption {
	return func(runtime *appRuntime) {
		runtime.binder.multipartMemory = memory
	}
}

// WithDescribedBy 为 ohmyhttp 写出的响应添加 RFC 8288 Link describedby 头，通常
// 指向 /openapi.json。
func WithDescribedBy(url string) AppOption {
	return func(runtime *appRuntime) {
		runtime.encoder.DescribedBy = url
	}
}

// WithMaxBodyBytes 设置请求体硬限制；非正数表示不限制。
func WithMaxBodyBytes(limit int64) AppOption {
	return func(runtime *appRuntime) {
		runtime.maxBodyBytes = limit
	}
}
