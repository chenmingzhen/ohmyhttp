package ohmyhttp

import (
	"context"
	"io"
	"io/fs"
	"net/http"
)

// Response HTTP 响应描述。
type Response struct {
	Status int
	Header http.Header
	Body   any
}

// FileBody 描述由 File 或 Attachment 创建的文件响应。
type FileBody struct {
	fileSystem   fs.FS
	name         string
	downloadName string
}

// Name 返回文件系统中的文件名。
func (body FileBody) Name() string { return body.name }

// DownloadName 返回附件下载名；内联文件响应返回空字符串。
func (body FileBody) DownloadName() string { return body.downloadName }

// RedirectBody 描述由 Redirect 创建的重定向响应。
type RedirectBody struct {
	location string
}

// Location 返回重定向目标。
func (body RedirectBody) Location() string { return body.location }

// RawBody 描述由 Text、HTML 或 Bytes 创建的原始字节响应。
type RawBody struct {
	contentType string
	data        []byte
}

// ContentType 返回原始响应的媒体类型。
func (body RawBody) ContentType() string { return body.contentType }

// Data 返回响应字节的副本。
func (body RawBody) Data() []byte { return append([]byte(nil), body.data...) }

// StreamWriter 以增量方式写入响应。当底层服务器支持 http.Flusher 时，flush 会把
// 缓冲区中的字节发送给客户端。
type StreamWriter func(context.Context, io.Writer, func() error) error

// StreamingBody 描述由 Streaming 或 EventStream 创建的流式响应。
type StreamingBody struct {
	write       StreamWriter
	contentType string
}

// ContentType 返回流式响应的媒体类型。
func (body StreamingBody) ContentType() string { return body.contentType }

// JSON 创建显式的 JSON 响应。
func JSON(status int, body any) Response {
	return Response{
		Status: status,
		Header: http.Header{
			"Content-Type": {jsonContentType},
		},
		Body: body,
	}
}

// Text 创建纯文本响应。
func Text(status int, text string) Response {
	return Bytes(status, "text/plain; charset=utf-8", []byte(text))
}

// HTML 创建 HTML 响应。
func HTML(status int, html string) Response {
	return Bytes(status, "text/html; charset=utf-8", []byte(html))
}

// Bytes 创建指定 Content-Type 的原始字节响应。
func Bytes(status int, contentType string, data []byte) Response {
	return Response{
		Status: status,
		Body: RawBody{
			contentType: contentType,
			data:        append([]byte(nil), data...),
		},
	}
}

// WithHeader 返回带有指定响应头的响应副本。
func (response Response) WithHeader(name string, values ...string) Response {
	if response.Header == nil {
		response.Header = http.Header{}
	} else {
		response.Header = response.Header.Clone()
	}
	response.Header.Del(name)
	for _, value := range values {
		response.Header.Add(name, value)
	}
	return response
}

// WithCookie 返回带有 Set-Cookie 响应头的响应副本。
func (response Response) WithCookie(cookie *http.Cookie) Response {
	if cookie == nil {
		return response
	}
	if response.Header == nil {
		response.Header = http.Header{}
	} else {
		response.Header = response.Header.Clone()
	}
	response.Header.Add("Set-Cookie", cookie.String())
	return response
}

// NoContent 创建 204 响应。
func NoContent() Response {
	return Response{Status: http.StatusNoContent}
}

// Redirect 创建不依赖具体路由器的重定向响应。
func Redirect(status int, location string) Response {
	return Response{
		Status: status,
		Body:   RedirectBody{location: location},
	}
}

// File 创建基于 fs.FS 的内联文件响应。
func File(fileSystem fs.FS, name string) Response {
	return Response{
		Status: http.StatusOK,
		Body: FileBody{
			fileSystem: fileSystem,
			name:       name,
		},
	}
}

// Attachment 创建基于 fs.FS 的附件下载响应。
func Attachment(fileSystem fs.FS, name string, downloadName string) Response {
	if downloadName == "" {
		downloadName = name
	}
	response := File(fileSystem, name)
	response.Body = FileBody{
		fileSystem:   fileSystem,
		name:         name,
		downloadName: downloadName,
	}
	return response
}

// Streaming 创建基于回调的流式响应。回调负责控制数据块边界，并可在每个数据块后
// 主动刷新。
func Streaming(status int, contentType string, writer StreamWriter) Response {
	return Response{
		Status: status,
		Body: StreamingBody{
			write:       writer,
			contentType: contentType,
		},
	}
}

// EventStream 创建 Server-Sent Events 响应。
func EventStream(writer StreamWriter) Response {
	response := Streaming(
		http.StatusOK,
		"text/event-stream; charset=utf-8",
		writer,
	)
	response.Header = http.Header{
		"Cache-Control": {"no-cache"},
	}
	return response
}
