# 错误处理

业务函数返回 `error` 时，框架把它写成 JSON，而不是把 `err.Error()` 原样发给客户端。默认正文类型是 `Problem`。

```go
type Problem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Details any    `json:"details,omitempty"`
}
```

## 返回业务错误

需要让客户端看到明确的状态码和业务错误时，使用 `NewError`。它会同时设置 HTTP 状态码、应用错误码和客户端可见的说明。

```go
type ReportRequest struct {
	ID string `path:"id" validate:"required"`
}

type Report struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

func GetReport(_ context.Context, request *ReportRequest) (*Report, error) {
	if request.ID != "monthly" {
		return nil, ohmyhttp.NewError(http.StatusNotFound, "report_not_found", "报表不存在").
			WithDetails(map[string]string{"id": request.ID})
	}
	return &Report{ID: request.ID, Name: "Monthly"}, nil
}

app.Get("/reports/{id}", GetReport)
```

`GET /reports/missing` 的响应：

```http
HTTP/1.1 404 Not Found
Content-Type: application/json; charset=utf-8

{"code":"report_not_found","message":"报表不存在","details":{"id":"missing"}}
```

`WithDetails` 返回带详情的副本。`HTTPStatus` 使用 400 到 599 的状态码。

如果需要封装自己的业务错误类型，让它实现 `HTTPError` 即可。接口包含 `HTTPStatus`、`ErrorCode`、`PublicMessage` 和 `ErrorDetails` 四个方法。

## 默认映射

| 返回的错误                                        | HTTP | 响应                                                         |
| ------------------------------------------------- | ---- | ------------------------------------------------------------ |
| `NewError(404, "report_not_found", "报表不存在")` | 404  | `code` 为 `report_not_found`                                 |
| 校验失败                                          | 400  | `code` 为 `validation_failed`，`details` 是字段列表          |
| 请求体超过 `MaxBodyBytes`                         | 413  | `code` 为 `request_too_large`                                |
| 不支持的 `Content-Type`                           | 415  | `code` 为 `unsupported_media_type`                           |
| JSON 无法解码                                     | 400  | `code` 为 `invalid_request`                                  |
| 其他未知错误                                      | 500  | `code` 为 `internal_error`，消息固定为 internal server error |

## 自定义映射

下面把校验失败改成 422，其他错误仍是 500。

```go
type mapper struct{}

func (mapper) MapError(_ context.Context, err error) ohmyhttp.Response {
	var validation *ohmyhttp.ValidationError
	if errors.As(err, &validation) {
		return ohmyhttp.JSON(http.StatusUnprocessableEntity, ohmyhttp.Problem{
			Code:    "validation_failed",
			Message: "request validation failed",
			Details: validation.Details,
		})
	}
	return ohmyhttp.JSON(http.StatusInternalServerError, ohmyhttp.Problem{
		Code:    "internal_error",
		Message: "internal server error",
	})
}

app := ohmyhttp.New(ohmyhttp.WithErrorMapper(mapper{}))
```

校验失败时的响应：

```http
HTTP/1.1 422 Unprocessable Entity

{"code":"validation_failed","message":"request validation failed","details":[...]}
```

## 记录服务端调用栈

内部错误默认只返回统一的 `internal_error`，不会把错误详情发给客户端。需要在日志中定位错误时，可以用可选的 `ohmyhttp/errorx` 在返回错误的位置保存调用栈：

```go
func LoadReport(ctx context.Context, request *ReportRequest) (*Report, error) {
	report, err := store.Find(ctx, request.ID)
	if err != nil {
		return nil, errorx.WithStack(err)
	}
	return report, nil
}
```

客户端仍收到 `internal_error`，服务端日志可以读取调用栈：

```go
ohmyhttp.WithErrorObserver(func(_ context.Context, err error, response ohmyhttp.Response) {
	if stack := errorx.StackTrace(err); stack != "" {
		log.Printf("status=%d\n%s", response.Status, stack)
	}
})
```
