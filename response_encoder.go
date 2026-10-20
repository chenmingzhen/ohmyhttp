package ohmyhttp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

// defaultEncoder 把结构化响应编码为 JSON，并直接处理显式的文本、HTML、字节、
// 文件、重定向和流式响应。
type defaultEncoder struct {
	DescribedBy string

	// protoJSON 只用于实现 proto.Message 的响应体。
	// 普通结构体仍走 encoding/json，不受这里影响。
	//
	// 未调用 WithProtoJSON 时，这里是 UseProtoNames: true。
	// WithProtoJSON 会把传入的 protojson.MarshalOptions 整组赋过来，
	// 包括把 UseProtoNames 设回 false 的零值。
	protoJSON protojson.MarshalOptions
}

func newDefaultEncoder() *defaultEncoder {
	return &defaultEncoder{
		// protojson 的零值把字段名写成 lowerCamelCase。未配置时改用 .proto
		// 里的字段名，避免 user_id 被写成 userId。未赋值字段默认省略。
		protoJSON: protojson.MarshalOptions{UseProtoNames: true},
	}
}

func (e *defaultEncoder) Encode(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	response Response,
) error {
	status := response.Status
	if status == 0 {
		status = http.StatusOK
	}
	if status < 100 || status > 999 {
		return fmt.Errorf("ohmyhttp: invalid response status %d", status)
	}

	copyHeaders(w.Header(), response.Header)
	if e.DescribedBy != "" && w.Header().Get("Link") == "" {
		w.Header().Add("Link", fmt.Sprintf("<%s>; rel=\"describedby\"", e.DescribedBy))
	}

	switch body := response.Body.(type) {
	case FileBody:
		return encodeFile(w, r, status, body)
	case RedirectBody:
		return encodeRedirect(w, r, status, body)
	case RawBody:
		return encodeRaw(w, r, status, body)
	case StreamingBody:
		return encodeStream(ctx, w, r, status, body)
	}

	if response.Body == nil || r.Method == http.MethodHead || statusDisallowsBody(status) {
		w.WriteHeader(status)
		return nil
	}

	return e.encodeJSON(w, status, response.Body)
}

func (e *defaultEncoder) encodeJSON(
	w http.ResponseWriter,
	status int,
	body any,
) error {
	var (
		data []byte
		err  error
	)
	// protobuf message 不能交给 encoding/json。生成代码里的 json 标签只覆盖
	// 一部分字段，enum、bytes、Timestamp 和字段是否出现都会和 protojson 不一致。
	if message, ok := body.(proto.Message); ok {
		data, err = e.protoJSON.Marshal(message)
	} else {
		data, err = json.Marshal(body)
	}
	if err != nil {
		return fmt.Errorf("marshal JSON response: %w", err)
	}

	w.Header().Set("Content-Type", jsonContentType)
	w.WriteHeader(status)
	_, err = w.Write(data)
	return err
}

func copyHeaders(destination http.Header, source http.Header) {
	for key, values := range source {
		for _, value := range values {
			destination.Add(key, value)
		}
	}
}

func statusDisallowsBody(status int) bool {
	return status >= 100 && status < 200 || status == http.StatusNoContent ||
		status == http.StatusNotModified
}
