# ohmyhttp

`ohmyhttp` 是构建在 Go `net/http` 上的 HTTP 框架。路由使用标准库的 `http.ServeMux`。`App` 实现 `http.Handler`。

端点签名为 `func(context.Context, *Req) (Resp, error)`。框架在调用端点前完成请求绑定和校验，在端点返回后完成错误映射和响应编码。已注册端点可用于生成 OpenAPI。

需要 Go 1.27 或更高版本。安装和最小示例见 [快速开始](/guide/getting-started)。

## 包

| 包 | 内容 |
| --- | --- |
| `ohmyhttp` | 应用、路由、绑定、校验、响应、错误、中间件、静态文件 |
| `ohmyhttp/meta` | 路由元数据 |
| `ohmyhttp/openapi` | 根据已注册端点生成 OpenAPI |
| `ohmyhttp/errorx` | 可选的服务端调用栈记录 |
| `ohmyhttp/server` | 监听、超时和进程退出 |

`examples/demo` 是一个可运行的程序，其中的鉴权和访问日志不是本库导出的 API。见 [示例](/reference/examples)。
