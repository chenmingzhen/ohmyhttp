package ohmyhttp

import (
	"fmt"
	"net/http"
	"reflect"

	"google.golang.org/protobuf/proto"
)

// bindRequestParameters 在 Request Body 解码之后，把 URL、Header 和路由中的
// 参数继续写入同一个 input。结构体通过 query、header、path 标签声明来源。
func bindRequestParameters(
	request *http.Request,
	requestBodyMediaType string,
	input any,
) error {
	// protobuf 字段由 protojson 按 protobuf descriptor 处理，不使用 Go
	// struct 上的 query/path/header/form 标签。
	if _, ok := input.(proto.Message); ok {
		return nil
	}
	return bindStructParameters(
		reflect.ValueOf(input),
		request,
		requestBodyMediaType,
	)
}

func bindStructParameters(
	target reflect.Value,
	request *http.Request,
	requestBodyMediaType string,
) error {
	// 先清空有来源标签的字段，确保它们只接受声明来源中的值。例如带
	// path:"id" 的字段只能由路径参数赋值，不能被 JSON Body 中的 id 冒充。
	if err := clearTaggedFields(target, "query", "path", "header"); err != nil {
		return fmt.Errorf("prepare request parameters: %w", err)
	}
	if isJSONMediaType(requestBodyMediaType) {
		if err := clearTaggedFields(target, "form"); err != nil {
			return fmt.Errorf("prepare form parameters: %w", err)
		}
	}

	// 按 Query -> Header -> Path 的顺序逐类写入字段。正常情况下一个字段只应
	// 声明一种来源；如果错误地声明多个来源，靠后的来源会覆盖靠前的值。
	// 它们修改的仍然是 Request Body 解码时使用的那个对象。
	if err := bindStructValues(target, "query", request.URL.Query()); err != nil {
		return fmt.Errorf("bind query: %w", err)
	}
	if err := bindStructHeaders(target, request.Header); err != nil {
		return fmt.Errorf("bind headers: %w", err)
	}
	if err := bindStructPathValues(target, request); err != nil {
		return fmt.Errorf("bind path: %w", err)
	}
	return nil
}

func bindStructPathValues(target reflect.Value, request *http.Request) error {
	return walkTaggedFields(target, "path", func(name string, field reflect.Value) error {
		value := request.PathValue(name)
		if value == "" {
			return nil
		}
		return setTextValues(field, []string{value})
	})
}
