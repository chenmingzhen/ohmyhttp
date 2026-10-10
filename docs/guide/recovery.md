# Panic 恢复

默认不恢复 panic。需要时在创建应用时启用：

```go
func Boom(context.Context, *struct{}) (struct{}, error) {
	panic("boom")
}

func main() {
	app := ohmyhttp.New(ohmyhttp.WithRecovery())
	app.Get("/boom", Boom)
}
```

`GET /boom` 的响应：

```http
HTTP/1.1 500 Internal Server Error
Content-Type: application/json; charset=utf-8

{"code":"internal_error","message":"internal server error"}
```

panic 的值和堆栈保存在 `*PanicError` 里，可以通过错误观察器记录，不会写进响应正文。外层 HTTP 中间件在返回后能读到这个 500。

## 自定义 panic 响应

```go
app := ohmyhttp.New(ohmyhttp.WithPanicMapper(func(request *http.Request, panicErr *ohmyhttp.PanicError) ohmyhttp.Response {
	log.Printf("panic %s %s: %v\n%s", request.Method, request.URL.Path, panicErr.Value, panicErr.StackTrace())
	return ohmyhttp.JSON(http.StatusInternalServerError, ohmyhttp.Problem{
		Code:    "panic",
		Message: "internal server error",
	})
}))
```

`WithPanicMapper` 同时打开 Recovery。返回的状态码为 0 时，框架写成 500。映射结果仍会经过端点响应转换和观察器。

如果响应在 panic 之前已经提交，例如流式接口写到一半，框架不再写入第二份响应，只做观察。

## 中止连接

`panic(http.ErrAbortHandler)` 会从 Recovery 里再次抛出。标准库据此关闭连接，不生成 HTTP 响应，这次 panic 也不会变成 `PanicError`。
