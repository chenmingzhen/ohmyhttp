package main

import (
	"log"

	"github.com/chenmingzhen/ohmyhttp/examples/demo"
	httpserver "github.com/chenmingzhen/ohmyhttp/server"
)

func main() {
	app := demo.New()

	const address = "127.0.0.1:8080"
	if err := httpserver.Run(app, httpserver.WithAddress(address)); err != nil {
		log.Fatal(err)
	}
}
