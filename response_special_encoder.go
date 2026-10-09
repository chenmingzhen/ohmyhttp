package ohmyhttp

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"mime"
	"net/http"
	"path"
	"strconv"
	"strings"
	"time"
)

// encodeRaw 写出内存中的原始字节响应。
func encodeRaw(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	body RawBody,
) error {
	if body.contentType != "" && w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", body.contentType)
	}
	w.WriteHeader(status)
	// HEAD 以及 1xx、204、304 只有状态行和响应头，不能再写正文。
	if r.Method == http.MethodHead || statusDisallowsBody(status) {
		return nil
	}
	_, err := w.Write(body.data)
	return err
}

// encodeRedirect 写出重定向响应。
func encodeRedirect(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	body RedirectBody,
) error {
	if body.location == "" {
		return errors.New("ohmyhttp: redirect location is empty")
	}
	if status < 300 || status > 399 {
		return fmt.Errorf("ohmyhttp: redirect requires a 3xx status, got %d", status)
	}
	http.Redirect(w, r, body.location, status)
	return nil
}

// encodeStream 写出由回调增量生成的响应体。
func encodeStream(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	status int,
	body StreamingBody,
) error {
	if body.write == nil {
		return errors.New("ohmyhttp: streaming writer is nil")
	}
	if body.contentType != "" && w.Header().Get("Content-Type") == "" {
		w.Header().Set("Content-Type", body.contentType)
	}
	w.WriteHeader(status)
	if r.Method == http.MethodHead || statusDisallowsBody(status) {
		return nil
	}
	// 第三个参数是 flush。CSV、SSE 每写出一块就调用它，把缓冲立刻发给客户端。
	return body.write(ctx, w, func() error {
		return flushHTTPWriter(w)
	})
}

// flushHTTPWriter 把已写入的响应缓冲立刻发到连接上。
func flushHTTPWriter(writer http.ResponseWriter) error {
	if err := http.NewResponseController(writer).Flush(); err != nil {
		return fmt.Errorf("ohmyhttp: flush response: %w", err)
	}
	return nil
}

// encodeFile 把 fs.FS 中的文件写入响应。
func encodeFile(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	body FileBody,
) error {
	if body.fileSystem == nil {
		return errors.New("ohmyhttp: file system is nil")
	}
	if !fs.ValidPath(body.name) {
		return fmt.Errorf("ohmyhttp: invalid file path %q", body.name)
	}
	file, err := body.fileSystem.Open(body.name)
	if err != nil {
		return fmt.Errorf("open file %q: %w", body.name, err)
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return fmt.Errorf("stat file %q: %w", body.name, err)
	}
	if info.IsDir() {
		return fmt.Errorf("file path %q is a directory", body.name)
	}
	if body.downloadName != "" {
		// 只保留文件名，避免下载名里的目录被写进 Content-Disposition。
		downloadName := path.Base(strings.ReplaceAll(body.downloadName, "\\", "/"))
		disposition := mime.FormatMediaType("attachment", map[string]string{
			"filename": downloadName,
		})
		if disposition == "" {
			return fmt.Errorf("format attachment filename %q", body.downloadName)
		}
		w.Header().Set("Content-Disposition", disposition)
	}
	if w.Header().Get("Content-Type") == "" {
		if contentType := mime.TypeByExtension(path.Ext(body.name)); contentType != "" {
			w.Header().Set("Content-Type", contentType)
		}
	}
	// ServeContent 会自己写出 200、206 或 304，不能用来保留调用方指定的其他状态码。
	if status != http.StatusOK {
		w.WriteHeader(status)
		if r.Method == http.MethodHead || statusDisallowsBody(status) {
			return nil
		}
		// 从 file 读到 EOF，写入响应体。内部用固定缓冲搬运，不把整个文件读进 []byte。
		_, err = io.Copy(w, file)
		return err
	}

	// ServeContent 依赖 Seek 处理 Range 和 If-Modified-Since。不能 Seek 时只能顺序读。
	seeker, ok := file.(io.ReadSeeker)
	if !ok {
		return encodeReadOnlyFile(w, r, file, info)
	}
	http.ServeContent(
		w,
		r,
		path.Base(body.name),
		info.ModTime().Truncate(time.Second),
		seeker,
	)
	return nil
}

// encodeReadOnlyFile 按顺序写出不能 Seek 的文件。
func encodeReadOnlyFile(
	w http.ResponseWriter,
	r *http.Request,
	file fs.File,
	info fs.FileInfo,
) error {
	reader := bufio.NewReader(file)
	if w.Header().Get("Content-Type") == "" {
		// Peek 把探测到的字节留在 reader 里，后面的 Copy 仍能从文件开头写出。
		prefix, err := reader.Peek(512)
		if err != nil && !errors.Is(err, io.EOF) {
			return fmt.Errorf("inspect file content: %w", err)
		}
		w.Header().Set("Content-Type", http.DetectContentType(prefix))
	}
	if w.Header().Get("Content-Length") == "" && info.Size() >= 0 {
		w.Header().Set("Content-Length", strconv.FormatInt(info.Size(), 10))
	}
	if modified := info.ModTime(); !modified.IsZero() && w.Header().Get("Last-Modified") == "" {
		w.Header().Set("Last-Modified", modified.UTC().Format(http.TimeFormat))
	}
	w.WriteHeader(http.StatusOK)
	if r.Method == http.MethodHead {
		return nil
	}
	// 读 reader 而不是 file：前缀还在缓冲区里，直接读 file 会丢掉已经 Peek 的字节。
	_, err := io.Copy(w, reader)
	return err
}
