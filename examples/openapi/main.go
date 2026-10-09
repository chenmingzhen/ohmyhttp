package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/chenmingzhen/ohmyhttp/examples/demo"
	"github.com/chenmingzhen/ohmyhttp/openapi"
)

func main() {
	format := flag.String("format", "json", "输出格式：json 或 yaml")
	output := flag.String("out", "", "输出文件路径；为空时输出到 stdout")
	version := flag.String("openapi-version", "", "OpenAPI 版本，例如 3.1.0 或 3.0.3")
	check := flag.Bool("check", false, "检查输出文件是否和当前路由一致，不写入文件")
	flag.Parse()

	options := demo.OpenAPIOptions()
	if strings.TrimSpace(*version) != "" {
		options = append(options, openapi.WithOpenAPIVersion(*version))
	}

	spec, err := openapi.Build(demo.New(), options...)
	if err != nil {
		fmt.Fprintf(os.Stderr, "build openapi: %v\n", err)
		os.Exit(1)
	}

	var data []byte
	switch strings.ToLower(*format) {
	case "json":
		data, err = openapi.JSON(spec)
	case "yaml", "yml":
		data, err = openapi.YAML(spec)
	default:
		err = fmt.Errorf("unsupported format %q", *format)
	}
	if err != nil {
		fmt.Fprintf(os.Stderr, "render openapi: %v\n", err)
		os.Exit(1)
	}
	data = append(data, '\n')

	if *check {
		if *output == "" {
			fmt.Fprintln(os.Stderr, "-check requires -out")
			os.Exit(1)
		}
		current, err := os.ReadFile(*output)
		if err != nil {
			fmt.Fprintf(os.Stderr, "read openapi file: %v\n", err)
			os.Exit(1)
		}
		if !bytes.Equal(current, data) {
			fmt.Fprintf(os.Stderr, "%s is out of date, run without -check to regenerate it\n", *output)
			os.Exit(1)
		}
		return
	}

	if *output != "" {
		if err := os.WriteFile(*output, data, 0o644); err != nil {
			fmt.Fprintf(os.Stderr, "write openapi file: %v\n", err)
			os.Exit(1)
		}
		return
	}

	if _, err := os.Stdout.Write(data); err != nil {
		fmt.Fprintf(os.Stderr, "write openapi: %v\n", err)
		os.Exit(1)
	}
}
