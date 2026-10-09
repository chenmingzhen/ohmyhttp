package server

import (
	"fmt"
	"net/http"
	"os"

	"github.com/chenmingzhen/ohmyhttp"
)

const developmentBanner = "  \033[1;38;2;66;211;146mohmyhttp\033[0m " +
	"\033[38;2;134;239;172m%s\033[0m" +
	" \033[90m· Listening and serving \033[0m" +
	"\033[1;38;2;134;239;172m%s\033[0m" +
	"\033[90m on \033[0m" +
	"\033[1;38;2;167;243;208m%s\033[0m"

type environmentProvider interface {
	Environment() ohmyhttp.Environment
}

func printDevelopmentBanner(handler http.Handler, protocol string, address string) {
	provider, ok := handler.(environmentProvider)
	if !ok {
		return
	}
	environment := provider.Environment()
	if !environment.IsDevelopment() {
		return
	}
	fmt.Fprintf(
		os.Stdout,
		developmentBanner+
			" \033[90m· Press \033[0m"+
			"\033[1;37mCtrl+C\033[0m\033[90m to stop\033[0m\n",
		environment,
		protocol,
		address,
	)
}
