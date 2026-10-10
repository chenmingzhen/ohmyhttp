# 中间件

框架有两层中间件，执行时机不同。

## HTTP 中间件

`Middleware` 就是 `func(http.Handler) http.Handler`。`Use` 追加到应用上，覆盖之后注册的全部请求：强类型端点、`ResponseFunc`、静态文件、原生 Handler 和 404/405。

```go
func requestID(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		id := r.Header.Get("X-Request-ID")
		if id == "" {
			id = "generated"
		}
		w.Header().Set("X-Request-ID", id)
		next.ServeHTTP(w, r)
	})
}

app.Use(requestID)
```

`GET /ping` 的响应会带上这个头。客户端没有发送 `X-Request-ID` 时，值是 `generated`。

```http
HTTP/1.1 200 OK
X-Request-ID: generated
```

`ChainMiddleware` 按声明顺序包装。第一个中间件在最外层：最先看到请求，最后看到响应。启用 Panic 恢复后，外层中间件可以记录到 panic 对应的最终状态码。

中间件里可以用 `ResponseInfoFromWriter` 查看内层已经写下的状态码。见 [可观测性](/guide/observability#响应写入快照)。

鉴权、CORS 和访问日志都写成这种中间件，由应用自己实现。

## 端点中间件

`EndpointMiddleware` 在解码和校验之后、业务函数之前运行。绑定失败或校验失败的请求到不了这里。

```go
type EndpointMiddleware func(*EndpointContext, EndpointNext) (Response, error)
```

三个注册点，执行时从外到内：

1. `app.UseEndpointMiddleware(...)` 包裹全部强类型端点
2. `group.UseEndpointMiddleware(...)` 包裹该分组及子分组之后注册的端点。父分组在外，子分组在内
3. `ohmyhttp.WithEndpointMiddleware(...)` 只包裹这一条路由

`ResponseFunc`、静态文件和 `HandleHTTP` 不进入端点中间件。

```go
func RequireUser(ctx *ohmyhttp.EndpointContext, next ohmyhttp.EndpointNext) (ohmyhttp.Response, error) {
	user, ok := ctx.Request.Context().Value(userKey{}).(User)
	if !ok {
		return ohmyhttp.Response{}, ohmyhttp.NewError(http.StatusUnauthorized, "unauthenticated", "未登录")
	}
	if user.Role != "admin" && ctx.Route.Pattern == "/admin" {
		return ohmyhttp.Response{}, ohmyhttp.NewError(http.StatusForbidden, "forbidden", "权限不足")
	}
	return next(ctx)
}
```

返回的 `error` 走统一错误映射。返回的 `Response` 在 `error == nil` 时进入编码。

## 端点上下文

`EndpointContext` 带有原始 `*http.Request`、匹配到的 `RouteInfo`，以及已绑定的请求值。

```go
input, ok := ohmyhttp.InputAs[*CreateUserRequest](ctx)
```

`InputAs` 读取已经绑定好的请求。`SetContext` 用来替换 `context.Context`，框架放进上下文的路由信息会保留。校验已经完成，所以这里读到的是通过校验的请求值。
