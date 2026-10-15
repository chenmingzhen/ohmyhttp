# 示例应用

`examples/demo` 演示 `ohmyhttp` 的完整请求生命周期。

- `app.go` 注册强类型端点、`ResponseFunc` 路由、OpenAPI 处理函数、静态资源和 `/browse/` 目录列表示例。
- `middleware.go` 演示请求 ID 和 CORS。
- `middleware.go` 也演示应用级 `UseEndpointMiddleware`。端点日志在参数绑定和校验之后运行，可以查看已经绑定的输入。
- `auth.go` 演示 JWT 解析、公开路由排除、授权回调、上下文注入和令牌刷新。
- `handler.go` 包含强类型处理函数和 Response 辅助方法。
- `websocket.go` 包含一个使用 HTTP Hijack 的原生 WebSocket 回显处理函数。
- `errors.go` 演示如何返回业务错误、记录服务端调用栈，以及记录包含 `method`、`path`、`rid`、`req`、`resp`、`code`、`error`、`cost` 字段的请求日志。
- `error_examples.go` 包含业务错误、带堆栈的内部错误、未知错误、panic 和 `http.ErrAbortHandler` 路由。
- `access_filter.go` 演示按路由转换和排除访问日志。
- `validation.go` 包含自定义校验规则。
- `openapi.go` 包含文档配置。

登录接口会签发示例 JWT。JWT 中间件保护 `/profile` 和 `/admin`，其余示例路由配置为公开路由。

在仓库根目录运行：

```powershell
go run ./examples/server
```

服务监听 `http://127.0.0.1:8080`。接口文档在 `http://127.0.0.1:8080/docs`。
