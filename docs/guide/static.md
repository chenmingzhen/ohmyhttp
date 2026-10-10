# 静态文件与 SPA

`Static` 和 `StaticDir` 把某个 URL 前缀下的 GET 请求对应到文件。`Static` 接受 `fs.FS`，`StaticDir` 接受本地目录。在分组上调用时，前缀会再叠上分组前缀。找不到文件时返回 JSON 404，错误码是 `static_not_found`。

```go
app.StaticDir("/assets", "public",
	ohmyhttp.StaticDirectoryListing(false),
	ohmyhttp.StaticCacheControl("public, max-age=3600"),
)

app.StaticDir("/browse", "files",
	ohmyhttp.StaticDirectoryListing(true),
)
```

## 选项

| 选项 | 行为 |
| --- | --- |
| `StaticIndexFile` | 目录请求使用的索引文件，默认 `index.html` |
| `StaticDirectoryListing` | 是否列出目录。默认关闭；关闭时目录没有索引就返回 404 |
| `StaticCacheControl` | 成功文件响应的 `Cache-Control` |
| `StaticSPAFile` | 找不到文件且 `Accept` 包含 `text/html` 时返回的页面，状态码仍是 200 |
| `StaticNotFoundFile` | 资源不存在时返回的文件，状态码改为 404 |

## 查找顺序

1. 请求路径对应目录本身时，返回索引文件
2. 文件不存在：先尝试 SPA 回退，再尝试 404 文件，最后返回 JSON 404
3. 命中目录：有索引文件就返回索引；允许列表时返回一段 HTML 目录；没有索引时返回 404
4. 命中文件：按 `File` 输出，并带上缓存头

目录列表按名字排序，目录链接带斜杠，名称经过 HTML 转义。

读取文件时的非「不存在」错误会作为内部错误返回，而不是伪装成 404。

## SPA

前端文件可以和 API 注册在同一个 `App` 上。页面导航和接口要分开处理，否则未知的 API 路径会返回首页：

```go
//go:embed public
var frontend embed.FS

publicFiles, _ := fs.Sub(frontend, "public")

api := app.Group("/api")
api.Get("/message", message)
api.HandleResponse(http.MethodGet, "/{path...}", apiNotFound)

app.Static("/", publicFiles, ohmyhttp.StaticSPAFile("index.html"))
```

`/api/message` 比 `/api/{path...}` 更具体，仍由端点处理。其他 `/api/*` 返回 JSON `api_not_found`，不会落到首页。浏览器访问未知页面时，`Accept` 含 `text/html`，静态路由返回 `index.html`。可运行代码在 `examples/spa`。
