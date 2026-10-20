# 应用选项

选项在 `ohmyhttp.New` 时生效，修改的是应用运行时，而不是某一条路由。路由级选项见文末。

## 错误与响应

| 选项 | 作用 |
| --- | --- |
| `WithErrorMapper(ErrorMapper)` | 替换默认错误映射 |
| `WithEndpointResponseTransformer` | 编码前转换强类型端点的结构化响应 |
| `WithNotFound(ResponseFunc)` | 替换 JSON 404 |
| `WithMethodNotAllowed(MethodNotAllowedFunc)` | 替换 JSON 405。未设置 `Allow` 时框架补上 |
| `WithDescribedBy(url)` | 为框架写出的响应添加 `Link: <url>; rel="describedby"`，已有 `Link` 时不覆盖 |
| `WithProtoJSON(protojson.MarshalOptions)` | 设置 protobuf 响应的 JSON 编码。见 [Protobuf](/guide/responses#protobuf) |
| `WithRecovery()` | 恢复 panic，并走错误映射 |
| `WithPanicMapper(PanicMapper)` | 恢复 panic，并用给定函数生成响应 |

## 观测

| 选项 | 调用时机 |
| --- | --- |
| `WithRequestObserver` | 强类型端点绑定和校验之后，失败也会调用 |
| `WithErrorObserver` | 错误映射和响应转换之后 |
| `WithResponseObserver` | 托管响应编码之后。原生 Handler 的正常响应不经过这里 |

## 请求体

| 选项 | 默认 | 说明 |
| --- | --- | --- |
| `WithMaxBodyBytes(limit)` | 32 MiB | 应用级上限。设为 0 表示不限制。路由级 `MaxBodyBytes` 优先 |
| `WithDiscardUnknownFields(bool)` | `false` | `false` 时拒绝 JSON 中的未知字段 |
| `WithProtoValidator` | 不校验 | 只校验实现 `proto.Message` 的请求。普通结构体仍用 `validate` 标签 |
| `WithMultipartMemory(memory)` | 8 MiB | multipart 解析留在内存中的最大字节数 |
| `WithValidationRule(tag, rule, ...)` | | 注册 validator 字段规则 |

## 环境

| 选项 | 说明 |
| --- | --- |
| `WithEnvironment(Environment)` | 直接设置环境。默认 `production` |
| `WithEnvironmentFromEnv(keys...)` | 从环境变量读取。不传键名时依次看 `OHMYHTTP_ENV`、`APP_ENV`、`GO_ENV`、`ENV` |

## 路由级选项

这些选项传给 `Get`、`Post` 等方法，只影响当前端点。

| 选项 | 说明 |
| --- | --- |
| `Status(code)` | 普通成功响应的状态码，默认 200 |
| `MaxBodyBytes(limit)` | 该路由的请求体上限。设为 0 表示不限制 |
| `WithEndpointMiddleware(...)` | 绑定和校验之后、业务函数之前执行 |
| `Describe(...)` | 写入 `RouteMetadata`，见 [路由元数据](/reference/metadata) |

`Describe` 也可以用于 `HandleResponse`。`Status`、`MaxBodyBytes` 和端点中间件只存在于强类型端点。

## 静态文件选项

传给 `Static` 和 `StaticDir`：`StaticIndexFile`、`StaticDirectoryListing`、`StaticCacheControl`、`StaticSPAFile`、`StaticNotFoundFile`。行为见 [静态文件与 SPA](/guide/static)。

## 服务器选项

`server.Run` 和 `RunTLS` 使用另一组选项，定义在 `ohmyhttp/server`：`WithAddress`、`WithReadHeaderTimeout`、`WithReadTimeout`、`WithWriteTimeout`、`WithIdleTimeout`、`WithShutdownTimeout`。默认值见 [HTTP 服务器](/guide/server)。

## OpenAPI 选项

定义在 `ohmyhttp/openapi`：`WithInfo`、`WithDescription`、`WithServer`、`WithOpenAPIVersion`、`WithSchemaName`、`WithSchemaCustomizer`、`WithSecurityScheme`、`WithGlobalSecurity`。文档页面还有 `WithDocsTitle` 和 `WithDocsAssets`。见 [OpenAPI](/guide/openapi)。
