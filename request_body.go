package ohmyhttp

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"reflect"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// decodeRequestBody 根据 Content-Type 选择请求体解码方式，并把结果写入 input。
// input 是调用方已经创建好的 *Req，因此这里所有解码操作都是原地修改。
func (b *requestBinder) decodeRequestBody(
	request *http.Request,
	requestBodyMediaType string,
	input any,
) error {
	// 调用方已经确认请求体存在。媒体类型只负责选择一种解码器；每个分支
	// 最终都会修改传入的 input，而不是创建并返回另一个请求对象。
	switch {
	case isJSONMediaType(requestBodyMediaType):
		// application/json 和 application/*+json。
		return b.decodeJSONRequestBody(request, input)
	case requestBodyMediaType == formMediaType:
		// application/x-www-form-urlencoded，例如 grant_type=password。
		return decodeURLFormRequestBody(request, input)
	case requestBodyMediaType == multipartMediaType:
		// multipart/form-data，既可以包含文本字段，也可以包含上传文件。
		return b.decodeMultipartRequestBody(request, input)
	default:
		// 请求体存在但没有对应解码器，包括缺少 Content-Type 的情况。
		return &UnsupportedMediaTypeError{MediaType: requestBodyMediaType}
	}
}

func (b *requestBinder) decodeJSONRequestBody(
	request *http.Request,
	input any,
) error {
	// 先把 Body 全部读成 []byte。请求大小限制已经在进入 Bind 前通过
	// http.MaxBytesReader 包装到 request.Body 上，因此这里不会绕过限制。
	data, err := readRequestBody(request)
	if err != nil {
		return err
	}
	if len(bytes.TrimSpace(data)) == 0 {
		return nil
	}
	// 使用 protobuf 类型来实现 HTTP/JSON 接口
	if message, ok := input.(proto.Message); ok {
		return decodeProtoJSON(data, message, b.discardUnknown)
	}

	// encoding/json 接收一个指针，并直接给指针指向的字段赋值。
	return decodeStrictJSON(data, input, b.discardUnknown)
}

// decodeProtoJSON 使用 protobuf 官方 JSON 映射解码请求体。protobuf message
// 不能复用 encoding/json，否则 enum、bytes、Any、Timestamp 和字段 presence
// 等语义都可能与 protobuf JSON 规范不一致。
func decodeProtoJSON(data []byte, destination proto.Message, discardUnknown bool) error {
	decoder := protojson.UnmarshalOptions{DiscardUnknown: discardUnknown}
	if err := decoder.Unmarshal(data, destination); err != nil {
		return fmt.Errorf("decode proto JSON: %w", err)
	}
	return nil
}

func decodeURLFormRequestBody(request *http.Request, input any) error {
	// ParseForm 会解析表单。这里使用 PostForm 而不是 Form，避免把 URL Query
	// 混进请求体阶段；Query 会由 bindRequestParameters 单独处理。
	if err := request.ParseForm(); err != nil {
		return fmt.Errorf("parse form: %w", err)
	}
	// 普通 struct 按结构体 form:"field_name" 标签查找字段，再把文本转换成字段类型。
	return bindStructValues(reflect.ValueOf(input), "form", request.PostForm)
}

func (b *requestBinder) decodeMultipartRequestBody(request *http.Request, input any) error {
	// ParseMultipartForm 把较小内容保存在内存，超过 memory 的文件部分放到
	// 临时文件；请求结束时 EndpointContext.cleanup 会删除这些临时文件。
	memory := b.multipartMemory
	if memory <= 0 {
		memory = defaultMultipartMemory
	}
	if err := request.ParseMultipartForm(memory); err != nil {
		return fmt.Errorf("parse multipart form: %w", err)
	}
	// 先绑定普通文本字段，再把上传文件绑定到 *multipart.FileHeader、
	// multipart.FileHeader 或 []*multipart.FileHeader 字段。
	if err := bindStructValues(
		reflect.ValueOf(input),
		"form",
		request.MultipartForm.Value,
	); err != nil {
		return err
	}
	return bindMultipartFiles(reflect.ValueOf(input), request.MultipartForm.File)
}

// requestBodyAvailable 判断是否有值得解码的请求体：
//
//   - Body 为 nil/http.NoBody 或 Content-Length 为 0：没有 Body；
//   - Content-Length 大于 0：有 Body，不提前读取；
//   - Content-Length 为 -1：长度未知，常见于 chunked 请求，探测一个字节。
//
// 长度未知时使用 bufio.Reader.Peek。Peek 只把字节放进缓冲区，不丢弃它；
// 随后再把这个 reader 装回 request.Body，所以真正的解码器仍能读到完整内容。
func requestBodyAvailable(request *http.Request) (bool, error) {
	if request.Body == nil || request.Body == http.NoBody || request.ContentLength == 0 {
		return false, nil
	}
	if request.ContentLength > 0 {
		return true, nil
	}

	reader := bufio.NewReader(request.Body)
	if _, err := reader.Peek(1); err != nil {
		if errors.Is(err, io.EOF) {
			return false, nil
		}
		return false, fmt.Errorf("read request body: %w", err)
	}
	request.Body = &bufferedRequestBody{Reader: reader, Closer: request.Body}
	return true, nil
}

// bufferedRequestBody 实现 io.ReadCloser 接口 让后面的流程继续从头开始读取字节
type bufferedRequestBody struct {
	io.Reader
	io.Closer
}

var _ io.ReadCloser = (*bufferedRequestBody)(nil)

func readRequestBody(request *http.Request) ([]byte, error) {
	// Body 是一次性流。Decode 只读取一次，然后 JSON 解码器处理得到的
	// []byte；后续参数绑定不会再次读取 Body。
	data, err := io.ReadAll(request.Body)
	if err != nil {
		return nil, fmt.Errorf("read request body: %w", err)
	}
	return data, nil
}

func decodeStrictJSON(data []byte, destination any, discardUnknown bool) error {
	// destination 是调用方创建的 *Req。json.Decoder.Decode 会直接修改
	// 这个指针指向的请求对象。
	decoder := json.NewDecoder(bytes.NewReader(data))
	if !discardUnknown {
		// 默认拒绝请求结构体中不存在的 JSON 字段，尽早暴露客户端拼写错误。
		decoder.DisallowUnknownFields()
	}
	if err := decoder.Decode(destination); err != nil {
		return fmt.Errorf("decode JSON: %w", err)
	}
	var extra any
	// 再尝试解一次，只允许得到 EOF，从而拒绝 `{...}{...}` 这种包含多个
	// 顶层 JSON 值的请求体。
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode JSON: multiple values are not allowed")
		}
		return fmt.Errorf("decode JSON trailing data: %w", err)
	}
	return nil
}
