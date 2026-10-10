# 注册路由

`App` 和 `Group` 都有同一组注册方法。方法名决定 HTTP 方法，模式使用 Go `net/http` 的路由语法。

```go
app.Get("/reports/{report_id}", GetReport)
app.Post("/reports", CreateReport)
app.Put("/reports/{report_id}", ReplaceReport)
app.Patch("/reports/{report_id}", UpdateReport)
app.Delete("/reports/{report_id}", DeleteReport)
app.Handle(http.MethodOptions, "/reports", OptionsReport)
```

`Handle` 用来注册上面五个方法以外的 HTTP 方法。

## 模式

| 模式 | 含义 |
| --- | --- |
| `/reports/{report_id}` | 单个路径段，业务里用 `path:"report_id"` 或 `PathParam` 读取 |
| `/files/{path...}` | 剩余全部路径 |
| `/{$}` | 只匹配站点根路径 `/`，不匹配更长的路径 |
| `/api/v1/ping` | 固定路径 |

更具体的模式优先于通配。例如 `/files/{path...}` 不会盖住已经注册的 `/files/readme`。

GET 路由同时接受 HEAD。HEAD 响应不包含正文。

## 路由分组

`Group` 给后续注册的路由加上路径前缀，并附带端点中间件。

```go
api := app.Group("/api")
v1 := api.Group("/v1")
v1.UseEndpointMiddleware(AuditTenant)
v1.Get("/ping", Ping)
```

最终模式是 `/api/v1/ping`。`prefix` 为空或 `"/"` 时不加前缀。子路由模式为 `"/"` 时，结果就是分组前缀本身。

`Group` 上也有 `HandleResponse`、`HandleHTTP`、`Static` 和 `StaticDir`。这些路由继承前缀。端点中间件只作用在强类型端点上，不会包住 `ResponseFunc` 和原生 Handler。

中间件顺序是 App、根分组、子分组、路由自身。详见 [中间件](/guide/middleware)。

## 状态码与请求体限制

```go
app.Post("/users", CreateUser,
	ohmyhttp.Status(http.StatusCreated),
	ohmyhttp.MaxBodyBytes(1<<20),
)
```

`Status` 是普通成功返回值使用的 HTTP 状态码，默认 200。端点若直接返回 `Response` 且状态码为 0，也会填上这个值。

`MaxBodyBytes` 只限制这一条路由。未设置时使用应用级限制，默认 32 MiB。超出后响应为 HTTP 413，错误码 `request_too_large`。设为 0 表示这条路由不限制请求体。

## 描述信息

```go
app.Get("/profile", Profile, ohmyhttp.Describe(
	meta.Name("profile"),
	meta.Summary("查询当前用户资料"),
	meta.Tags("user"),
	meta.Security(ohmyhttp.SecurityRequirement{"bearerAuth": nil}),
	meta.Returns(http.StatusOK, "用户资料", (*ProfileResponse)(nil)),
	meta.ReturnsError(http.StatusUnauthorized, "未登录或登录状态失效"),
))
```

`Describe` 写入 `RouteMetadata`。日志、指标和 OpenAPI 读取它。它不改变匹配，也不代替鉴权中间件。字段说明见 [路由元数据](/reference/metadata)。

## 回退

没有路由匹配时，客户端看到的是 JSON。

`GET /missing`：

```json
{"code":"not_found","message":"route not found"}
```

`POST /ping`，而 `/ping` 只注册了 GET：

```http
HTTP/1.1 405 Method Not Allowed
Allow: GET, HEAD

{"code":"method_not_allowed","message":"method not allowed"}
```

两种响应都可以替换：

```go
app := ohmyhttp.New(
	ohmyhttp.WithNotFound(func(*http.Request) (ohmyhttp.Response, error) {
		return ohmyhttp.JSON(http.StatusNotFound, ohmyhttp.Problem{
			Code:    "missing",
			Message: "这里没有这个接口",
		}), nil
	}),
	ohmyhttp.WithMethodNotAllowed(func(_ *http.Request, allowed []string) (ohmyhttp.Response, error) {
		return ohmyhttp.JSON(http.StatusMethodNotAllowed, ohmyhttp.Problem{
			Code:    "method_not_allowed",
			Message: "换一个方法",
		}).WithHeader("Allow", strings.Join(allowed, ", ")), nil
	}),
)
```

自定义 405 若没有自己写 `Allow`，框架会补上。

## 已注册端点

`app.Endpoints()` 返回已注册的强类型端点，供日志或 OpenAPI 使用。`HandleHTTP` 和静态文件不在这个列表里。如果配置了响应类型转换，这里的响应类型是转换之后的类型。
