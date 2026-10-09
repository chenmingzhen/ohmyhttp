# ohmyhttp

[Go Reference][go-reference] · [中文](README.md)

`ohmyhttp` is a type-safe HTTP application framework built directly on Go's `net/http`. It preserves the standard library's routing and Handler ecosystem while adding request binding, validation, error mapping, response encoding, and OpenAPI support.

## Features

- Path, query, header, cookie, form, and request body binding
- Struct-tag validation
- JSON, text, file, streaming, and redirect responses
- Centralized error mapping, endpoint response transformation, and encoding
- Request-level and endpoint-level middleware
- Route groups and OpenAPI generation

## Installation

Go 1.27 or later is required.

```bash
go get github.com/chenmingzhen/ohmyhttp
```

## Quick start

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

Start the server and open `GET /hello?name=World`.
