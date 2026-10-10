# 快速开始

需要 Go 1.27 或更高版本。

```bash
go get github.com/chenmingzhen/ohmyhttp
```

## 第一个端点

```go
package main

import (
	"context"
	"log"

	"github.com/chenmingzhen/ohmyhttp"
	httpserver "github.com/chenmingzhen/ohmyhttp/server"
)

type HelloRequest struct {
	Name string `query:"name" validate:"required"`
}

type HelloResponse struct {
	Message string `json:"message"`
}

func Hello(_ context.Context, request *HelloRequest) (HelloResponse, error) {
	return HelloResponse{Message: "Hello, " + request.Name}, nil
}

func main() {
	app := ohmyhttp.New()
	app.Get("/hello", Hello)

	if err := httpserver.Run(app); err != nil {
		log.Fatal(err)
	}
}
```

`httpserver.Run` 默认监听 `127.0.0.1:8080`。访问 `GET /hello?name=World` 得到：

```json
{"message":"Hello, World"}
```

漏掉 `name` 时，校验失败，响应是 HTTP 400：

```json
{
  "code": "validation_failed",
  "message": "request validation failed",
  "details": [
    {"field": "name", "rule": "required"}
  ]
}
```

## 运行示例应用

仓库自带一个覆盖主要特性的示例。在仓库根目录执行：

```powershell
go run ./examples/server
```

服务监听 `http://127.0.0.1:8080`。接口文档在 `http://127.0.0.1:8080/docs`，OpenAPI JSON 在 `/openapi.json`。请求样本在 `examples/requests.http`。

另外三个小例子：

| 目录 | 演示 |
| --- | --- |
| `examples/spa` | `embed.FS` 托管前端，API 未命中时返回 JSON，页面导航回退到 `index.html`。监听 `127.0.0.1:8081` |
| `examples/http200` | 响应转换器把业务错误改成 HTTP 200，业务码放在 JSON 里 |
| `examples/openapi` | 离线生成或检查 OpenAPI 文件 |

## 本地预览这份文档

```bash
cd docs
pnpm install
pnpm docs:dev
```

下一页说明请求从进入 `App` 到写出响应的顺序。
