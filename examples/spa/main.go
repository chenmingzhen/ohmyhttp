package main

import (
	"context"
	"embed"
	"io/fs"
	"log"
	"net/http"

	"github.com/chenmingzhen/ohmyhttp"
	httpserver "github.com/chenmingzhen/ohmyhttp/server"
)

//go:embed public
var frontend embed.FS

type messageResponse struct {
	Message string `json:"message"`
}

func main() {
	publicFiles, err := fs.Sub(frontend, "public")
	if err != nil {
		log.Fatal(err)
	}

	app := ohmyhttp.New(
		ohmyhttp.WithEnvironment(ohmyhttp.EnvDevelopment),
	)
	api := app.Group("/api")
	api.Get("/message", message)
	api.HandleResponse(http.MethodGet, "/{path...}", apiNotFound)

	app.Static(
		"/",
		publicFiles,
		ohmyhttp.StaticSPAFile("index.html"),
	)

	if err := httpserver.Run(
		app,
		httpserver.WithAddress("127.0.0.1:8081"),
	); err != nil {
		log.Fatal(err)
	}
}

func message(
	_ context.Context,
	_ *struct{},
) (messageResponse, error) {
	return messageResponse{Message: "Hello from the ohmyhttp API."}, nil
}

func apiNotFound(*http.Request) (ohmyhttp.Response, error) {
	return ohmyhttp.JSON(http.StatusNotFound, ohmyhttp.Problem{
		Code:    "api_not_found",
		Message: "API route not found",
	}), nil
}
