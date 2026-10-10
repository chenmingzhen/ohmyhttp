# OpenAPI

`openapi` 包根据 `App.Endpoints()` 生成 OpenAPI 3.1.0 或 3.0.3，只包含强类型端点。`HandleHTTP` 和静态文件不会自动出现；需要出现在文档里时，改成强类型端点，或在元数据里声明响应。

## 运行时入口

```go
options := []openapi.Option{
	openapi.WithOpenAPIVersion("3.1.0"),
	openapi.WithInfo("My API", "0.1.0"),
	openapi.WithDescription("由已注册端点生成的 OpenAPI 文档。"),
	openapi.WithServer("http://127.0.0.1:8080", "本地"),
	openapi.WithSecurityScheme("bearerAuth", openapi.SecurityScheme{
		Type:         "http",
		Scheme:       "bearer",
		BearerFormat: "JWT",
	}),
}

app.HandleResponse(http.MethodGet, "/openapi.json", openapi.JSONHandler(app, options...))
app.HandleResponse(http.MethodGet, "/openapi.yaml", openapi.YAMLHandler(app, options...))
app.HandleResponse(http.MethodGet, "/schemas", openapi.SchemasHandler(app, options...))
app.HandleResponse(http.MethodGet, "/docs", openapi.DocsHandler("/openapi.json",
	openapi.WithDocsTitle("My API"),
))
```

| Handler | 结果 |
| --- | --- |
| `JSONHandler` / `SpecHandler(..., FormatJSON)` | `application/json` 的完整文档 |
| `YAMLHandler` | `application/yaml` |
| `SchemasHandler` | `components.schemas`。查询参数 `name` 指定某一个；不存在时 404，错误码 `schema_not_found` |
| `DocsHandler` | 内置的 Swagger UI 页面，默认从 jsDelivr 加载静态资源 |

`WithDocsAssets` 可换成自建的 CSS 和 JS 地址。`specURL` 为空时使用 `/openapi.json`。

这些 Handler 返回 `ResponseFunc`，因此享受统一编码和观察，本身不进入 `Endpoints()`，不会把自己再写进生成的文档。

## 文档里有什么

每条强类型路由变成一个 operation：

- `operationId`、摘要、描述、废弃标记、标签来自 `meta`
- `path`、`query`、`header` 标签变成参数。路径参数在文档里始终必填；其他参数在带 `validate:"required"` 时必填
- POST、PUT、PATCH 上，带 `json` 标签的字段组成 `application/json` 请求体；带 `form` 标签的字段组成表单。没有二进制字段时同时给出 `application/x-www-form-urlencoded` 和 `multipart/form-data`，有文件字段时只给出 multipart
- 成功状态码和响应类型来自路由。声明了 `meta.Returns` 时，以声明为准
- `meta.Security` 写成 operation 级 security。`WithGlobalSecurity` 写在文档顶层
- `meta.Extension` 的 `x-` 键进入 operation 扩展。`meta.Value` 留在运行时元数据里，不会自动进 OpenAPI

Go 模式会改写成 OpenAPI 路径：`/files/{path...}` 变成 `/files/{path}`，结尾的 `{$}` 会被去掉。

## Schema

具名结构体默认进入 `components.schemas`，名称是包名加类型名。例如 `user.LoginRequest` 得到 `UserLoginRequest`。`WithSchemaName` 可以改这个名称。

| Go 类型 | Schema |
| --- | --- |
| `bool` | boolean |
| 整数 | integer |
| 浮点 | number |
| `string` | string |
| `[]byte` | string / binary |
| `time.Time` | string / date-time |
| `multipart.FileHeader` | string / binary |
| 切片、数组 | array |
| `map` | object + additionalProperties |
| 指针 | `nullable: true` |

字段上的文档标签：

| 标签 | 作用 |
| --- | --- |
| `description` 或 `doc` | 字段说明 |
| `format` | 覆盖 format |
| `pattern` | 正则 |
| `default` | 默认值，按字段类型解析字面量 |
| `example` | 示例值 |
| `enum` | 逗号或空白分隔的枚举 |
| `nullable:"true"` | 可空。指针字段也会标可空 |
| `readOnly` / `writeOnly` | 对应布尔标记 |

`validate` 标签会翻译成约束：`email`、`uri`、`uuid`、`datetime`、`hostname`、`ip` / `ipv4` / `ipv6`、`oneof`、`min`、`max`、`gte`、`lte`、`gt`、`lt`、`len`。`required` 进入对象的 `required` 数组。

类型可以实现 `OpenAPISchema() *Schema`，完全接管自己的 schema。`WithSchemaCustomizer` 在生成之后收到这份 schema，可以改 `pattern`、`example` 或 `description`。

`WithOpenAPIVersion("3.0.3")` 会把 3.1 的可空写法归一到 3.0 能表达的形式。

## 安全方案

```go
openapi.WithSecurityScheme("bearerAuth", openapi.SecurityScheme{
	Type:         "http",
	Scheme:       "bearer",
	BearerFormat: "JWT",
	Description:  "登录后签发的令牌。",
})
```

路由上：

```go
meta.Security(ohmyhttp.SecurityRequirement{"bearerAuth": nil})
```

这只写入 OpenAPI。请求是否放行，由应用自己的中间件决定。

## describedby

```go
ohmyhttp.WithDescribedBy("/openapi.json")
```

响应还没有 `Link` 头时，会追加：

```http
Link: </openapi.json>; rel="describedby"
```

这是 RFC 8288 的关系类型，用来指向描述该响应的文档。

## 离线生成

不启动服务也可以生成文档：

```go
spec, err := openapi.Build(app, options...)
if err != nil {
	return err
}
data, err := openapi.JSON(spec)
```

YAML 使用 `openapi.YAML`。仓库里的 `examples/openapi` 把这个调用包成命令行，`-check` 只比较输出是否与已有文件一致。
