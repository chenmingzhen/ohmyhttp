# 示例

`examples/demo` 把主要用法放在一个服务里。在仓库根目录：

```powershell
go run ./examples/server
```

监听 `http://127.0.0.1:8080`。Swagger UI 在 `/docs`。样本请求在 `examples/requests.http`。

应用在 `New` 里打开了这些开关：Recovery、开发环境、请求/错误/响应观察、`describedby` 指向 `/openapi.json`、忽略未知 JSON 字段、multipart 内存 16 MiB，以及自定义 `username` 规则。

HTTP 中间件依次是请求 ID、访问日志、CORS、JWT。端点中间件是一条执行日志，能看到绑定后的输入类型。

## 示例里的路由

JWT 保护 `GET /profile`、`PATCH /profile` 和 `GET /admin`。登录和其他示例路由在中间件的排除列表里。

登录相关：

- `POST /tenants/{tenant_id}/login` 同时绑定路径参数和 JSON，成功后签发示例 JWT。
- `POST /oauth/token` 读取 URL 编码表单，并用请求级校验区分密码模式和刷新令牌模式。
- `POST /sessions` 返回响应头和 `Set-Cookie`，这条路由的请求体上限是 512 字节。
- `GET /admin` 在授权回调里拒绝名为 `disabled` 的用户。

用户和报表：

- `GET /profile` 从查询参数读取页码。`PATCH /profile` 从请求头读取用户 ID，从 JSON 读取要改的资料。这两条和 `GET /admin` 都要带登录令牌。
- `POST /users` 使用自定义用户名规则，并检查两次密码是否一致。
- `POST /users/{user_id}/avatar` 绑定 multipart 的文本字段和文件，请求体上限 32 MiB。
- `POST /tenants/{tenant_id}/reports` 同时绑定路径、请求头和 JSON，成功状态码是 201。
- `GET /reports/search` 把重复的查询参数绑定成字符串切片。
- `GET /reports/{report_id}/metadata` 直接返回响应，并写出 `ETag` 和 `Cache-Control`。
- `GET /reports/{report_id}/download` 下载附件，`/redirect` 做跳转，`/export` 按行刷出 CSV。
- `DELETE /reports/{report_id}` 返回 204。
- `GET /events` 是服务端事件流。

其余路径用来看框架边上的行为：

- `GET /` 只匹配站点根路径，返回一段 HTML。`GET /api/v1/ping`、`/api/v1/hello.html`、`/api/v1/bytes` 在两级分组下，分别返回 JSON、HTML 和原始字节。
- `GET /ws/echo` 是原生 Handler 上的 WebSocket 回显。
- `GET /healthz` 返回 204，并且不进访问日志。
- `GET /errors/business`、`/stack`、`/unknown`、`/panic`、`/abort` 分别对应公开错误、带堆栈的内部错误、普通未知错误、由 Recovery 恢复的 panic，以及中止连接。
- `GET /openapi.json`、`/openapi.yaml`、`/schemas`、`/docs` 提供文档。`/schemas` 可以用查询参数 `name` 取某一个组件。
- `GET /assets/*` 是带一小时缓存的静态目录。`GET /browse/` 会列出目录。

访问日志会按路由打码：`/tenants/*/login`、`/oauth/token`、`/sessions` 去掉 `password`，`/users` 再去掉 `confirm_password`。请求头和字段名里的 `access_token`、`refresh_token`、`authorization`、`token`、`secret` 也会被遮住。

## 另外三个程序

`go run ./examples/spa` 把前端嵌进二进制。`/api` 下找不到的路径返回 JSON，页面导航回退到 `index.html`，监听 `127.0.0.1:8081`。

`go run ./examples/http200` 用响应转换把业务错误改成 HTTP 200，业务码放在 JSON 里，同样监听 `127.0.0.1:8081`。

`go run ./examples/openapi -format json` 不启动服务，把示例应用的 OpenAPI 打到标准输出。

离线检查：

```powershell
go run ./examples/openapi -check -format json -out api.json
```

`-check` 要求同时给出 `-out`。文件与当前路由不一致时进程以非零状态退出。
