# 错误类型

端点返回的错误由 `DefaultMapError` 分类后写成响应。默认正文是 `Problem`。普通 `error` 的 `Error()` 默认不会直接写入响应。

## 内置错误

| 类型 | 何时产生 | 默认响应 |
| --- | --- | --- |
| `*Error` | `NewError`，实现 `HTTPError` | 声明的 4xx/5xx、`ErrorCode`、`ErrorMessage`、`ErrorDetails` |
| `*ValidationError` | 标签校验、`NewValidationError` | 400 `validation_failed`，`details` 为 `[]Violation` 或调用方给的详情 |
| `*RequestBindingError` | JSON、表单、query、header、path 绑定失败 | 400 `invalid_request`。`Unwrap` 得到原因，原因不写入响应 |
| `*UnsupportedMediaTypeError` | 有正文但没有可用解码器，或缺少 Content-Type | 415 `unsupported_media_type`。`Message` 使用该错误的 `Error()` |
| `*http.MaxBytesError` | 超出 `MaxBodyBytes` | 413 `request_too_large` |
| `*PanicError` | Recovery 恢复的 panic，包含恢复值和堆栈 | 500 `internal_error`。`Value` 不进入响应 |

404 `not_found` 和 405 `method_not_allowed` 在路由匹配失败时产生。处理函数返回的错误，以及请求绑定、校验和请求体读取产生的错误，都会转换为对应的错误响应。

静态文件找不到时，响应为 404 `static_not_found`。

## HTTPError

```go
type HTTPError interface {
	error
	HTTPStatus() int
	ErrorCode() string
	ErrorMessage() string
	ErrorDetails() any
}
```

`NewError(status, message, opts...)` 创建默认实现。`code` 默认来自状态码，例如 400 是 `bad_request`。`WithCode` 用来覆盖，`WithDetails` 用来设置详情。

## Violation

```go
type Violation struct {
	Field     string `json:"field"`
	Rule      string `json:"rule"`
	Parameter string `json:"parameter,omitempty"`
	Message   string `json:"message,omitempty"`
}
```

标签错误由框架从 validator 的字段错误转换。`ValidateRequest` 要携带说明时，返回 `NewValidationError(cause, details)`。

## 服务端调用栈

`ohmyhttp/errorx` 提供可选的调用栈记录能力。使用 `WithStack` 包装错误后，调用栈可以写入服务端日志，不会出现在客户端响应中。错误仍按原始类型生成响应；普通内部错误仍返回 500 `internal_error`。

| 函数 | 行为 |
| --- | --- |
| `WithStack(err)` | 在调用处保存堆栈。`nil` 或已经带有调用栈的错误会原样返回 |
| `StackTrace(err)` | 读取错误链中第一个 `StackError` 的格式化堆栈，没有时返回空字符串 |

`StackError` 的堆栈适合写入服务端日志，不应作为响应正文返回。Recovery 产生的 `PanicError` 由错误观察器直接读取。
