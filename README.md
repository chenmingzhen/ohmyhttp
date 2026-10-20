# ohmyhttp

[文档](https://chenmingzhen.github.io/ohmyhttp/) · [English](README_EN.md)

`ohmyhttp` 是一个直接构建在 Go `net/http` 之上的强类型 HTTP 应用框架。它保留标准库的路由和 Handler 生态，同时补齐请求绑定、校验、错误映射、响应编码与 OpenAPI。

## 核心能力

- 路径、查询参数、请求头、Cookie、表单和请求体绑定
- 基于结构体标签的参数校验
- JSON、文本、文件、流式响应和重定向
- 统一的错误映射、Endpoint 响应转换与响应编码
- 请求级和端点级中间件
- 路由分组与 OpenAPI 生成
- 可选地使用 protobuf 类型实现 HTTP/JSON 接口

## 安装

需要 Go 1.27 或更高版本。

```bash
go get github.com/chenmingzhen/ohmyhttp
```

## 快速开始

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

启动后访问 `GET /hello?name=World`。
