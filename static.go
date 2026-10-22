package ohmyhttp

import (
	"errors"
	"fmt"
	"html"
	"io/fs"
	"net/http"
	"net/url"
	"os"
	"path"
	"sort"
	"strings"
)

// StaticOption 配置静态资源路由。
type StaticOption func(*staticConfig)

// StaticIndexFile 设置目录请求使用的默认索引文件。
func StaticIndexFile(name string) StaticOption {
	return func(config *staticConfig) {
		config.indexFile = strings.TrimSpace(name)
	}
}

// StaticDirectoryListing 控制是否允许列出静态目录内容。
func StaticDirectoryListing(enabled bool) StaticOption {
	return func(config *staticConfig) {
		config.directoryBrowse = enabled
	}
}

// StaticCacheControl 设置静态文件响应的 Cache-Control 头。
func StaticCacheControl(value string) StaticOption {
	return func(config *staticConfig) {
		config.cacheControl = strings.TrimSpace(value)
	}
}

// StaticSPAFile 设置 HTML 导航请求找不到文件时使用的 SPA 文件。
func StaticSPAFile(name string) StaticOption {
	return func(config *staticConfig) {
		config.spaFile = strings.TrimSpace(name)
	}
}

// StaticNotFoundFile 设置静态资源不存在时返回的文件。
func StaticNotFoundFile(name string) StaticOption {
	return func(config *staticConfig) {
		config.notFoundFile = strings.TrimSpace(name)
	}
}

// Static 在 prefix 下注册由框架管理的静态文件目录。目录浏览默认关闭。
func (app *App) Static(prefix string, fileSystem fs.FS, options ...StaticOption) {
	registerStatic(app, prefix, fileSystem, options...)
}

// StaticDir 在 prefix 下注册本地静态文件目录。
func (app *App) StaticDir(prefix string, root string, options ...StaticOption) {
	app.Static(prefix, os.DirFS(root), options...)
}

// Static 在当前分组下注册静态文件目录。
func (group *Group) Static(prefix string, fileSystem fs.FS, options ...StaticOption) {
	hierarchy := group.configurationHierarchy()
	for _, item := range hierarchy {
		prefix = joinPath(item.prefix, prefix)
	}
	registerStatic(group.app, prefix, fileSystem, options...)
}

// StaticDir 在当前分组下注册本地静态文件目录。
func (group *Group) StaticDir(prefix string, root string, options ...StaticOption) {
	group.Static(prefix, os.DirFS(root), options...)
}

type staticConfig struct {
	indexFile       string
	directoryBrowse bool
	cacheControl    string
	spaFile         string
	notFoundFile    string
}

func registerStatic(
	app *App,
	prefix string,
	fileSystem fs.FS,
	options ...StaticOption,
) {
	prefix = normalizeGroupPrefix(prefix)
	if prefix == "" {
		prefix = "/"
	}
	config := staticConfig{indexFile: "index.html"}
	for _, option := range options {
		option(&config)
	}
	validateStaticFileName("index", config.indexFile, true)
	validateStaticFileName("SPA", config.spaFile, true)
	validateStaticFileName("not found", config.notFoundFile, true)

	pattern := "/{path...}"
	if prefix != "/" {
		pattern = prefix + "/{path...}"
	}
	app.registerResponseRoute(responseRoute{
		method:  http.MethodGet,
		pattern: pattern,
		kind:    RouteKindStatic,
		metadata: RouteMetadata{
			Name: "static:" + prefix,
		},
		respond: staticResponseFunc(fileSystem, config),
	})
}

func validateStaticFileName(kind string, name string, allowEmpty bool) {
	if name == "" && allowEmpty {
		return
	}
	if !fs.ValidPath(name) || name == "." {
		panic(fmt.Sprintf("ohmyhttp: invalid static %s file %q", kind, name))
	}
}

func staticResponseFunc(fileSystem fs.FS, config staticConfig) ResponseFunc {
	return func(request *http.Request) (Response, error) {
		name := strings.TrimPrefix(request.PathValue("path"), "/")
		if name == "" {
			name = "."
		}
		if !fs.ValidPath(name) {
			return Response{}, NewError(
				http.StatusNotFound,
				"static resource does not exist",
				WithCode("static_not_found"),
			)
		}

		info, err := fs.Stat(fileSystem, name)
		if err != nil {
			return staticMissingResponse(request, fileSystem, config, err)
		}
		if info.IsDir() {
			if config.indexFile != "" {
				indexName := path.Join(name, config.indexFile)
				if indexInfo, indexErr := fs.Stat(fileSystem, indexName); indexErr == nil &&
					!indexInfo.IsDir() {
					return staticFileResponse(fileSystem, indexName, config), nil
				}
			}
			if !config.directoryBrowse {
				return Response{}, NewError(
					http.StatusNotFound,
					"static resource does not exist",
					WithCode("static_not_found"),
				)
			}
			return staticDirectoryResponse(request, fileSystem, name)
		}
		return staticFileResponse(fileSystem, name, config), nil
	}
}

func staticMissingResponse(
	request *http.Request,
	fileSystem fs.FS,
	config staticConfig,
	err error,
) (Response, error) {
	if !isFileMissing(err) {
		return Response{}, fmt.Errorf("read static resource: %w", err)
	}
	if config.spaFile != "" && acceptsHTML(request) {
		if info, statErr := fs.Stat(fileSystem, config.spaFile); statErr == nil && !info.IsDir() {
			return staticFileResponse(fileSystem, config.spaFile, config), nil
		}
	}
	if config.notFoundFile != "" {
		if info, statErr := fs.Stat(fileSystem, config.notFoundFile); statErr == nil &&
			!info.IsDir() {
			response := staticFileResponse(fileSystem, config.notFoundFile, config)
			response.Status = http.StatusNotFound
			return response, nil
		}
	}
	return Response{}, NewError(
		http.StatusNotFound,
		"static resource does not exist",
		WithCode("static_not_found"),
	)
}

func staticFileResponse(fileSystem fs.FS, name string, config staticConfig) Response {
	response := File(fileSystem, name)
	if config.cacheControl != "" {
		response = response.WithHeader("Cache-Control", config.cacheControl)
	}
	return response
}

func staticDirectoryResponse(
	request *http.Request,
	fileSystem fs.FS,
	name string,
) (Response, error) {
	entries, err := fs.ReadDir(fileSystem, name)
	if err != nil {
		return Response{}, fmt.Errorf("read static directory: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })
	var output strings.Builder
	output.WriteString("<!doctype html><html><body><h1>Index of ")
	output.WriteString(html.EscapeString(request.URL.Path))
	output.WriteString("</h1><ul>")
	for _, entry := range entries {
		label := entry.Name()
		if entry.IsDir() {
			label += "/"
		}
		output.WriteString("<li><a href=\"")
		output.WriteString(url.PathEscape(entry.Name()))
		if entry.IsDir() {
			output.WriteByte('/')
		}
		output.WriteString("\">")
		output.WriteString(html.EscapeString(label))
		output.WriteString("</a></li>")
	}
	output.WriteString("</ul></body></html>")
	return HTML(http.StatusOK, output.String()), nil
}

func isFileMissing(err error) bool {
	return errors.Is(err, fs.ErrNotExist)
}

func acceptsHTML(request *http.Request) bool {
	return strings.Contains(
		strings.ToLower(request.Header.Get("Accept")),
		"text/html",
	)
}
