# 错误类型

默认客户端协议是 `Problem`。下表是框架和 `errorx` 定义的错误，以及默认 `ErrorMapper` 的结果。

## 框架错误

| 类型 | 何时产生 | 默认响应 |
| --- | --- | --- |
| `*Error` | `NewError`，实现 `PublicError` | 声明的 4xx/5xx、`ErrorCode`、`PublicMessage`、`ErrorDetails` |
| `*ValidationError` | 标签校验、`NewValidationError` | 400 `validation_failed`，`details` 为 `[]Violation` 或调用方给的详情 |
| `*RequestBindingError` | JSON、表单、query、header、path 绑定失败 | 400 `invalid_request`。`Unwrap` 得到原因，原因不写入响应 |
| `*UnsupportedMediaTypeError` | 有正文但没有可用解码器，或缺少 Content-Type | 415 `unsupported_media_type` |
| `*http.MaxBytesError` | 超出 `MaxBodyBytes` | 413 `request_too_large` |
| `*PanicError` | Recovery 恢复的 panic，包含恢复值和堆栈 | 500 `internal_error` |
| `*errorx.StackError` | `errorx.WithStack` | 500 `internal_error` |

404 `not_found` 和 405 `method_not_allowed` 在路由匹配时产生，不会进入 `WithErrorMapper`。只有处理函数自己返回的错误才会走错误映射。

静态文件缺失使用公开错误：404 `static_not_found`。

## PublicError

```go
type PublicError interface {
	error
	HTTPStatus() int
	ErrorCode() string
	PublicMessage() string
	ErrorDetails() any
}
```

`NewError(status, code, message)` 创建默认实现。`WithDetails` 返回带详情的浅拷贝。

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

## errorx

| 函数 | 行为 |
| --- | --- |
| `WithStack(err)` | 在调用处保存堆栈 |
| `HasStack(err)` | 错误链上是否存在 `StackTracer` |
| `StackTrace(err)` | 第一段格式化堆栈，没有时返回空字符串 |

`StackTracer` 只有一个方法：`StackTrace() string`。`*StackError` 和 `*PanicError` 都实现了它。

堆栈文本的每一帧是函数名，以及文件和行号。它面向日志，默认响应正文不包含这段文本。
