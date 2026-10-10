# 路由元数据

`ohmyhttp/meta` 里的选项只描述路由。它们用于日志、指标和 OpenAPI，不改变路由匹配，也不执行鉴权。

通过 `ohmyhttp.Describe` 一次性传入：

```go
app.Post("/users", CreateUser, ohmyhttp.Describe(
	meta.Name("create_user"),
	meta.Summary("创建用户"),
	meta.Description("用户名使用自定义 username 规则，密码需要二次确认。"),
	meta.Tags("user"),
	meta.Deprecated(),
	meta.Value("audit", true),
	meta.Extension("x-demo", map[string]string{"feature": "validation"}),
	meta.Security(ohmyhttp.SecurityRequirement{"bearerAuth": nil}),
	meta.Returns(http.StatusCreated, "创建成功", (*CreateUserResponse)(nil)),
	meta.ReturnsError(http.StatusBadRequest, "请求参数错误"),
))
```

## 字段

| 选项 | 写入 | OpenAPI |
| --- | --- | --- |
| `Name` | 稳定名称 | `operationId` |
| `Summary` | 短摘要 | `summary` |
| `Description` | 详细说明 | `description` |
| `Deprecated` | 废弃标记 | `deprecated: true` |
| `Tags` | 标签 | `tags` |
| `Value(key, value)` | `Values` 映射 | 不输出 |
| `Extension` | `x-` 开头的扩展，放进 `Values` | operation 扩展 |
| `Security` | security requirement 列表 | operation `security` |

`SecurityRequirement` 是 `map[string][]string`，键为方案名，值为 scope。文档专用。

## 返回值

未声明 `Returns` 时，OpenAPI 用路由成功状态码和响应类型生成一条 JSON 响应。声明之后，以声明列表为准。

| 选项 | 文档内容 |
| --- | --- |
| `Returns(status, description, body)` | `application/json`。`body` 传 `(*Type)(nil)` 或零值，框架取其类型 |
| `ReturnsError(status, description)` | JSON，正文类型为 `*Problem` |
| `ReturnsContent(status, description, contentType, body)` | 指定媒体类型 |
| `ReturnsFile(status, contentType, description)` | 二进制文件 |
| `ReturnsStream(status, contentType, description)` | 二进制流 |

1xx、204 和 304 在文档里不带正文，实际响应也同样没有正文。

响应转换器如果同时实现了类型转换，`Returns` 里可转换的 JSON 类型会换成线上外壳的类型。文件和流保持二进制声明。

## 读取

业务代码里用 `ohmyhttp.MetaFromContext` 拿到 `RouteInfo.Metadata`。`Endpoints()` 返回的是克隆后的快照：标签、返回值、安全要求和 `Values` 都已复制，调用方修改副本不会影响已注册路由。
