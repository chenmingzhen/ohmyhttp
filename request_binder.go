package ohmyhttp

import (
	"fmt"
	"mime"
	"net/http"
	"strings"
)

// requestBinder 把一条 HTTP 请求绑定到路由声明的请求对象。
//
// 它不会创建请求对象。每次收到请求时，调用方会先通过端点路由保存的工厂
// 创建一个空的 *Req，再把这个指针作为 input 传给 Bind。Bind 随后把
// Request Body、Query、Header 和 Path 参数写入同一个对象。
type requestBinder struct {
	multipartMemory int64
	discardUnknown  bool
}

func newRequestBinder(multipartMemory int64) *requestBinder {
	return &requestBinder{
		multipartMemory: multipartMemory,
	}
}

func (b *requestBinder) Bind(
	request *http.Request,
	input any,
) error {
	// 两个参数分别表示：
	//
	//   - request：net/http 路由匹配后的原始 HTTP 请求；
	//   - input：已经创建好的 *Req，整个绑定链路都在修改它。
	//
	// 例如 Req 为 CreateUserRequest 时，input 的动态类型是
	// *CreateUserRequest，初始值相当于 &CreateUserRequest{}。
	hasRequestBody, err := requestBodyAvailable(request)
	if err != nil {
		return err
	}

	requestBodyMediaType := ""
	if hasRequestBody {
		// Content-Type 只决定如何解释请求体。例如 application/json 使用 JSON
		// 解码。
		requestBodyMediaType, err = parseRequestBodyMediaType(request)
		if err != nil {
			return err
		}
		if err := b.decodeRequestBody(
			request,
			requestBodyMediaType,
			input,
		); err != nil {
			return err
		}
	}

	// 继续在同一个 input 上绑定 Query、Header 和 Path 参数。
	// 例如 query:"page"、header:"Authorization"、path:"id" 分别从 URL、
	// Header 和路由参数中取值。Bind 成功后，调用方会校验这个 input，
	// 然后把同一个指针交给业务 Endpoint。
	return bindRequestParameters(request, requestBodyMediaType, input)
}

// parseRequestBodyMediaType 去掉 Content-Type 中的参数，只保留小写的媒体类型。
// 例如 "application/json; charset=utf-8" 会得到 "application/json"。
func parseRequestBodyMediaType(request *http.Request) (string, error) {
	value := strings.TrimSpace(request.Header.Get("Content-Type"))
	if value == "" {
		return "", nil
	}
	mediaType, _, err := mime.ParseMediaType(value)
	if err != nil {
		return "", fmt.Errorf("parse Content-Type: %w", err)
	}
	return strings.ToLower(mediaType), nil
}
