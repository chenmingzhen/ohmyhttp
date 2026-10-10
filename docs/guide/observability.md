# 可观测性

三个观察器都是可选的，在 `ohmyhttp.New` 时设置。它们只读快照，不改写响应。观察器不应修改请求对象，也不应在请求结束后继续持有它。

## 请求观察器

```go
ohmyhttp.WithRequestObserver(func(ctx context.Context, observation ohmyhttp.RequestObservation) {
	log.Printf("%s %s err=%v", observation.Route.Method, observation.Route.Pattern, observation.Err)
})
```

调用点在强类型端点绑定和校验之后、端点中间件之前。`Input` 是请求对象。成功时 `Err` 为 nil。绑定或校验失败时仍会调用：`Input` 可能只填了一部分，`Err` 是 `*RequestBindingError` 或 `*ValidationError`。

`Route` 里有方法、路径模式和 `Describe` 写上的元数据。日志里可以用 `Name` 作为操作名，也可以读取 `Values` 里自己放入的标记，决定要不要省略某些字段。

## 错误观察器

```go
ohmyhttp.WithErrorObserver(func(ctx context.Context, err error, response ohmyhttp.Response) {
	log.Printf("status=%d err=%v", response.Status, err)
})
```

调用时，错误已经变成客户端将要看到的响应。`err` 仍是原始错误，适合记录堆栈或指标，不能再改这份响应。

## 响应观察器

```go
ohmyhttp.WithResponseObserver(func(
	ctx context.Context,
	route ohmyhttp.RouteInfo,
	response ohmyhttp.Response,
	err error,
) {
	log.Printf("%s %s status=%d", route.Method, route.Pattern, response.Status)
})
```

端点、`HandleResponse` 和静态文件写出响应后调用。成功时 `err` 为 nil。`HandleHTTP` 自己写出的正常响应不会出现在这里。

## 响应写入快照

在 HTTP 中间件里可以读取已经写出的状态码和字节数。

```go
info, ok := ohmyhttp.ResponseInfoFromWriter(writer)
if ok {
	log.Printf("status=%d bytes=%d committed=%v", info.StatusCode, info.BytesWritten, info.Committed)
}
```

尚未写出状态码时，`StatusCode` 按 200 报告。自己再包装 `ResponseWriter` 时，需要实现 `Unwrap() http.ResponseWriter`，否则读不到这份状态。

## 请求元数据

匹配之后，业务函数和端点中间件可以用上下文读到路由：

```go
meta, ok := ohmyhttp.MetaFromContext(ctx)
id := ohmyhttp.PathParam(ctx, "report_id")
```

`RequestMeta` 包含 `*http.Request` 和 `RouteInfo`。`PathParam` 在元数据缺失时返回空字符串。路径字段更常见的读法是 `path` 标签；`PathParam` 适合中间件和不想把参数放进结构体的代码。
