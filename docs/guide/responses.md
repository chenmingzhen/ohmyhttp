# 响应编码

业务函数返回结构体时，框架按成功状态码把它编码成 JSON，默认 200。要控制状态码、媒体类型或响应头时，返回 `ohmyhttp.Response`。

## JSON

```go
type PingResponse struct {
	Value string `json:"value"`
}

func Ping(context.Context, *struct{}) (PingResponse, error) {
	return PingResponse{Value: "pong"}, nil
}

app.Get("/ping", Ping)
```

`GET /ping` 的响应：

```http
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8

{"value":"pong"}
```

创建资源时用 `Status` 改成功状态码：

```go
app.Post("/users", CreateUser, ohmyhttp.Status(http.StatusCreated))
```

`POST /users` 成功时是 `201 Created`，正文仍是返回值的 JSON。

## 文本、HTML 和空响应

```go
func Message(context.Context, *struct{}) (ohmyhttp.Response, error) {
	return ohmyhttp.Text(http.StatusOK, "pong"), nil
}

func Page(context.Context, *struct{}) (ohmyhttp.Response, error) {
	return ohmyhttp.HTML(http.StatusOK, "<h1>ohmyhttp</h1>"), nil
}

func Remove(context.Context, *struct{}) (ohmyhttp.Response, error) {
	return ohmyhttp.NoContent(), nil
}
```

`Text` 的 `Content-Type` 是 `text/plain; charset=utf-8`。`HTML` 是 `text/html; charset=utf-8`。`NoContent` 的响应只有状态行：

```http
HTTP/1.1 204 No Content
```

`Bytes` 用来指定其他媒体类型：

```go
return ohmyhttp.Bytes(http.StatusOK, "application/octet-stream", data), nil
```

## 响应头和 Cookie

`WithHeader` 和 `WithCookie` 返回新的响应，不修改原值。

```go
func Login(_ context.Context, request *LoginRequest) (ohmyhttp.Response, error) {
	token := issueToken(request.Account)
	return ohmyhttp.JSON(http.StatusCreated, map[string]string{"token": token}).
		WithHeader("X-Request-ID", "req-1").
		WithCookie(&http.Cookie{
			Name:     "session",
			Value:    token,
			Path:     "/",
			HttpOnly: true,
		}), nil
}
```

```http
HTTP/1.1 201 Created
Content-Type: application/json; charset=utf-8
X-Request-ID: req-1
Set-Cookie: session=tok_demo; Path=/; HttpOnly

{"token":"tok_demo"}
```

`WithHeader` 会替换同名头。`WithCookie` 追加 `Set-Cookie`。传入 `nil` cookie 时响应不变。

## 重定向

```go
func GoToDownload(context.Context, *struct{}) (ohmyhttp.Response, error) {
	return ohmyhttp.Redirect(http.StatusFound, "/reports/monthly/download"), nil
}
```

```http
HTTP/1.1 302 Found
Location: /reports/monthly/download
```

状态码必须是 3xx，目标不能为空。

## 文件

```go
//go:embed files
var reportFiles embed.FS

func Download(context.Context, *struct{}) (ohmyhttp.Response, error) {
	return ohmyhttp.Attachment(reportFiles, "files/monthly.txt", "monthly.txt"), nil
}
```

`Attachment` 设置 `Content-Disposition: attachment`。`File` 不设置这个头，浏览器按文件类型直接打开。状态码 200 且文件可 Seek 时支持 Range 和 `If-Modified-Since`。

## 流和 SSE

```go
func ExportCSV(context.Context, *struct{}) (ohmyhttp.Response, error) {
	return ohmyhttp.Streaming(http.StatusOK, "text/csv; charset=utf-8", func(ctx context.Context, w io.Writer, flush func() error) error {
		for _, row := range []string{"id,name", "1,Ada"} {
			if err := ctx.Err(); err != nil {
				return err
			}
			if _, err := fmt.Fprintln(w, row); err != nil {
				return err
			}
			if err := flush(); err != nil {
				return err
			}
		}
		return nil
	}), nil
}
```

客户端按行收到：

```text
id,name
1,Ada
```

`EventStream` 的 `Content-Type` 是 `text/event-stream; charset=utf-8`，并设置 `Cache-Control: no-cache`。

```go
func Events(context.Context, *struct{}) (ohmyhttp.Response, error) {
	return ohmyhttp.EventStream(func(ctx context.Context, w io.Writer, flush func() error) error {
		if _, err := fmt.Fprint(w, "data: hello\n\n"); err != nil {
			return err
		}
		return flush()
	}), nil
}
```

流已经开始写出之后如果失败，不能再改状态码。

HEAD、1xx、204 和 304 不写正文。`WithDescribedBy("/openapi.json")` 会在还没有 `Link` 头时追加 `<url>; rel="describedby"`。
