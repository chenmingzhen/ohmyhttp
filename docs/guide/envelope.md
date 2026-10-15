# 响应转换

`EndpointResponseTransformer` 在错误映射之后、编码之前改写强类型端点的最终 `Response`。可以用来给成功和失败套同一层 JSON，也可以把部分业务错误改成 HTTP 200。

```go
app := ohmyhttp.New(
	ohmyhttp.WithEndpointResponseTransformer(
		ohmyhttp.EndpointResponseTransformFunc(func(
			_ context.Context,
			_ ohmyhttp.RouteInfo,
			response ohmyhttp.Response,
			err error,
		) ohmyhttp.Response {
			if err != nil {
				return response
			}
			response.Body = map[string]any{
				"success": true,
				"data":    response.Body,
			}
			return response
		}),
	),
)
```

`EndpointResponseTransformFunc` 把函数适配成接口。转换器能看到路由、映射后的响应，以及原始错误。返回值替换即将编码的响应。

## 哪些响应会进入转换

同时满足这些条件才会进入：

- 路由种类是 `endpoint`
- 状态码允许正文
- Body 不是文件、重定向、原始字节或流

因此 `HTML`、`Bytes`、`File`、`Redirect`、`Streaming`、`EventStream`、`NoContent` 保持原样。`HandleResponse` 和静态文件不经过这个转换器。

## 类型转换

线上外壳和函数返回类型不一致时，实现 `EndpointResponseTypeTransformer`，OpenAPI 会使用转换后的类型。

```go
func (APIResponseTransformer) TransformEndpointResponseType(
	status int,
	bodyType reflect.Type,
) reflect.Type {
	return reflect.TypeOf(APIResponse{})
}
```

`Endpoints()` 的 `ResponseType`，以及 `Metadata.Returns` 里可转换的 JSON 响应体，都会换成这个类型。文件、二进制和非 JSON 内容类型保持原声明。

## 业务错误使用 HTTP 200

有的接口希望传输成功时 HTTP 状态保持 200，业务成败放在 JSON 里。转换器可以在错误映射之后改状态码。下面的 `BusinessError` 让框架先生成业务错误响应，再统一改写成接口需要的格式：

```go
type BusinessError struct {
	code    int
	message string
}

func (e *BusinessError) Error() string         { return e.message }
func (e *BusinessError) HTTPStatus() int       { return http.StatusUnprocessableEntity }
func (e *BusinessError) ErrorCode() string     { return strconv.Itoa(e.code) }
func (e *BusinessError) PublicMessage() string { return e.message }
func (e *BusinessError) ErrorDetails() any     { return nil }

type APIResponse struct {
	Code       int  `json:"code"`
	Data       any  `json:"data"`
	Successful bool `json:"successful"`
}

func (APIResponseTransformer) TransformEndpointResponse(
	_ context.Context,
	_ ohmyhttp.RouteInfo,
	response ohmyhttp.Response,
	err error,
) ohmyhttp.Response {
	if err == nil {
		response.Body = APIResponse{Code: 0, Data: response.Body, Successful: true}
		return response
	}

	result := APIResponse{Code: response.Status * 100, Successful: false}
	var businessErr *BusinessError
	if errors.As(err, &businessErr) {
		response.Status = http.StatusOK
		result.Code = businessErr.code
	}
	response.Body = result
	return response
}
```

`BusinessError` 的 HTTP 状态是 422。转换器认出它之后改成 200，并把业务码写入 JSON 的 `code`。校验失败等其他错误仍保留 4xx，正文同样换成 `APIResponse`。
