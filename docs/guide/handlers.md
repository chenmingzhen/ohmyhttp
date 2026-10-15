# 处理函数

`App` 可以同时注册下面三种路由。

## 强类型端点

```go
type CreateUserRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type CreateUserResponse struct {
	ID string `json:"id"`
}

func CreateUser(_ context.Context, request *CreateUserRequest) (CreateUserResponse, error) {
	return CreateUserResponse{ID: "user_" + request.Email}, nil
}

app.Post("/users", CreateUser, ohmyhttp.Status(http.StatusCreated))
```

签名固定为 `func(context.Context, *Req) (Resp, error)`。`Req` 和 `Resp` 是类型参数，由 `Get`、`Post` 等方法推断。

返回值有三种形态：

- 普通值或指针：按 `Status` 编码。结构体默认成为 JSON
- `ohmyhttp.Response`：使用其中的状态码、头和 Body。状态码为 0 时改用路由的成功状态码
- `*ohmyhttp.Response`：同上。返回 `nil` 指针时，写出一个没有正文的成功状态码

业务错误返回 `error`。默认不会把 `err.Error()` 写给客户端，除非这个错误实现了 `HTTPError`。见 [错误处理](/guide/errors)。

## ResponseFunc

适合健康检查、OpenAPI、已经手写好响应描述的路由。

```go
app.HandleResponse(http.MethodGet, "/healthz", func(*http.Request) (ohmyhttp.Response, error) {
	return ohmyhttp.NoContent(), nil
})
```

处理函数读取 `*http.Request`，返回 `Response` 和 `error`。这里没有请求结构体，也不会执行端点中间件。错误仍然按统一规则返回给客户端。

## HandleHTTP

需要控制连接时使用原生 Handler：升级 WebSocket、Hijack、或者把已有 `http.Handler` 嵌进来。

```go
app.HandleHTTP(http.MethodGet, "/ws/echo", http.HandlerFunc(echo))
```

HTTP 中间件仍然会执行。响应由这个 `Handler` 自己写出，`ResponseObserver` 收不到这份正常响应。WebSocket 升级就注册在这里。

## 对应关系

| 需求 | 注册方式 |
| --- | --- |
| JSON API，请求字段要校验 | `Get` / `Post` / `Put` / `Patch` / `Delete` |
| 固定响应，或自己从 `*http.Request` 取值 | `HandleResponse` |
| 文档、健康检查、Swagger UI | `HandleResponse` |
| 上传后仍返回 JSON 结构体 | 强类型端点，字段用 `form` 和 `*multipart.FileHeader` |
| WebSocket、流式协议升级、遗留 Handler | `HandleHTTP` |
| 前端构建产物 | `Static` / `StaticDir` |

页面和 API 可以注册在同一个 `App` 上。更具体的路径仍由端点处理，同组下更宽的模式可以返回 JSON 404，页面则用静态文件回退到 `index.html`。见 [静态文件](/guide/static#spa)。
