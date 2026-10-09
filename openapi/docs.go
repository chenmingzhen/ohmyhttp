package openapi

import (
	"bytes"
	"html/template"
	"net/http"
	"strings"

	"github.com/chenmingzhen/ohmyhttp"
)

type DocsOption func(*docsConfig)

func WithDocsTitle(title string) DocsOption {
	return func(config *docsConfig) {
		if strings.TrimSpace(title) != "" {
			config.Title = title
		}
	}
}

func WithDocsAssets(cssURL string, jsURL string) DocsOption {
	return func(config *docsConfig) {
		if strings.TrimSpace(cssURL) != "" {
			config.CSS = cssURL
		}
		if strings.TrimSpace(jsURL) != "" {
			config.JS = jsURL
		}
	}
}

// DocsHandler 返回由 ohmyhttp 统一编码的 Swagger UI 响应函数。
func DocsHandler(specURL string, options ...DocsOption) ohmyhttp.ResponseFunc {
	specURL = strings.TrimSpace(specURL)
	if specURL == "" {
		specURL = "/openapi.json"
	}
	config := docsConfig{
		Title: defaultDocsTitle,
		CSS:   defaultDocsCSS,
		JS:    defaultDocsJS,
	}
	for _, option := range options {
		option(&config)
	}
	return func(_ *http.Request) (ohmyhttp.Response, error) {
		var output bytes.Buffer
		if err := docsHTML.Execute(&output, map[string]string{
			"Title": config.Title,
			"Spec":  specURL,
			"CSS":   config.CSS,
			"JS":    config.JS,
		}); err != nil {
			return ohmyhttp.Response{}, err
		}
		return ohmyhttp.HTML(http.StatusOK, output.String()), nil
	}
}

const (
	defaultDocsTitle = "API Docs"
	defaultDocsCSS   = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui.css"
	defaultDocsJS    = "https://cdn.jsdelivr.net/npm/swagger-ui-dist@5/swagger-ui-bundle.js"
)

type docsConfig struct {
	Title string
	CSS   string
	JS    string
}

var docsHTML = template.Must(template.New("docs").Parse(`<!doctype html>
<html lang="zh-CN">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <title>{{ .Title }}</title>
    <link rel="stylesheet" href="{{ .CSS }}">
    <style>
      body { margin: 0; background: #f7f7f7; }
      #swagger-ui { min-height: 100vh; }
    </style>
  </head>
  <body>
    <div id="swagger-ui"></div>
    <script src="{{ .JS }}"></script>
    <script>
      window.ui = SwaggerUIBundle({
        url: {{ .Spec }},
        dom_id: '#swagger-ui',
        deepLinking: true,
        presets: [SwaggerUIBundle.presets.apis]
      });
    </script>
  </body>
</html>`))
