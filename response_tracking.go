package ohmyhttp

import (
	"bufio"
	"io"
	"net"
	"net/http"
)

// trackedResponseWriter 记录响应状态，同时保留常见的可选 ResponseWriter 能力。
type trackedResponseWriter struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int64
	written      bool
}

var (
	_ http.ResponseWriter = (*trackedResponseWriter)(nil)
	_ http.Flusher        = (*trackedResponseWriter)(nil)
	_ http.Hijacker       = (*trackedResponseWriter)(nil)
	_ http.Pusher         = (*trackedResponseWriter)(nil)
	_ io.StringWriter     = (*trackedResponseWriter)(nil)
	_ io.ReaderFrom       = (*trackedResponseWriter)(nil)
	_ responseTracker     = (*trackedResponseWriter)(nil)
)

func (w *trackedResponseWriter) WriteHeader(statusCode int) {
	if w.written {
		return
	}
	w.written = true
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *trackedResponseWriter) Write(data []byte) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	n, err := w.ResponseWriter.Write(data)
	w.bytesWritten += int64(n)
	return n, err
}

func (w *trackedResponseWriter) WriteString(value string) (int, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	if writer, ok := w.ResponseWriter.(io.StringWriter); ok {
		n, err := writer.WriteString(value)
		w.bytesWritten += int64(n)
		return n, err
	}
	return w.Write([]byte(value))
}

func (w *trackedResponseWriter) ReadFrom(source io.Reader) (int64, error) {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	if reader, ok := w.ResponseWriter.(io.ReaderFrom); ok {
		n, err := reader.ReadFrom(source)
		w.bytesWritten += n
		return n, err
	}
	return io.Copy(struct{ io.Writer }{w}, source)
}

func (w *trackedResponseWriter) Flush() {
	if !w.written {
		w.WriteHeader(http.StatusOK)
	}
	_ = http.NewResponseController(w.ResponseWriter).Flush()
}

// 把底层 TCP 连接从 net/http 手中取走，后续由 Handler 完全管理
func (w *trackedResponseWriter) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	connection, readWriter, err := http.NewResponseController(w.ResponseWriter).Hijack()
	if err == nil && !w.written {
		w.written = true
		w.statusCode = http.StatusSwitchingProtocols
	}
	return connection, readWriter, err
}

func (w *trackedResponseWriter) Push(target string, options *http.PushOptions) error {
	for current := w.ResponseWriter; current != nil; {
		if pusher, ok := current.(http.Pusher); ok {
			return pusher.Push(target, options)
		}
		unwrapper, ok := current.(interface{ Unwrap() http.ResponseWriter })
		if !ok {
			break
		}
		next := unwrapper.Unwrap()
		if next == current {
			break
		}
		current = next
	}
	return http.ErrNotSupported
}

func (w *trackedResponseWriter) Written() bool { return w.written }

func (w *trackedResponseWriter) StatusCode() int { return w.statusCode }

func (w *trackedResponseWriter) BytesWritten() int64 { return w.bytesWritten }

// Unwrap 使 http.ResponseController 能访问原始 ResponseWriter 的可选能力。
func (w *trackedResponseWriter) Unwrap() http.ResponseWriter { return w.ResponseWriter }
