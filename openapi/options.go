package openapi

import (
	"reflect"
	"strings"
)

// Option 配置 OpenAPI 构建过程。
type Option func(*config)

// WithInfo 设置 OpenAPI info。
func WithInfo(title string, version string) Option {
	return func(config *config) {
		config.info.Title = title
		config.info.Version = version
	}
}

// WithDescription 设置 OpenAPI info.description。
func WithDescription(description string) Option {
	return func(config *config) {
		config.info.Description = description
	}
}

// WithServer 添加一个 server。
func WithServer(url string, description string) Option {
	return func(config *config) {
		config.servers = append(config.servers, Server{
			URL:         url,
			Description: description,
		})
	}
}

// WithOpenAPIVersion 设置输出的 OpenAPI 版本。常用值是 3.1.0 或 3.0.3。
func WithOpenAPIVersion(version string) Option {
	return func(config *config) {
		version = strings.TrimSpace(version)
		if version != "" {
			config.version = version
		}
	}
}

// WithSchemaName 自定义组件 schema 名称。返回空字符串表示该类型使用内联 schema。
func WithSchemaName(name func(reflect.Type) string) Option {
	return func(config *config) {
		config.schemaName = name
	}
}

// WithSchemaCustomizer 允许应用在生成后微调某个 Go 类型对应的 schema。
func WithSchemaCustomizer(customize func(reflect.Type, *Schema) *Schema) Option {
	return func(config *config) {
		config.schemaCustomize = customize
	}
}

// WithSecurityScheme 注册 OpenAPI components.securitySchemes。
func WithSecurityScheme(name string, scheme SecurityScheme) Option {
	return func(config *config) {
		name = strings.TrimSpace(name)
		if name == "" {
			return
		}
		if config.securitySchemes == nil {
			config.securitySchemes = map[string]SecurityScheme{}
		}
		config.securitySchemes[name] = scheme
	}
}

// WithGlobalSecurity 设置 OpenAPI 顶层 security requirement。
func WithGlobalSecurity(requirements ...SecurityRequirement) Option {
	return func(config *config) {
		config.security = cloneSecurity(requirements)
	}
}

type config struct {
	version         string
	info            Info
	servers         []Server
	schemaName      func(reflect.Type) string
	schemaCustomize func(reflect.Type, *Schema) *Schema
	securitySchemes map[string]SecurityScheme
	security        []SecurityRequirement
}

func newConfig(options []Option) config {
	config := config{
		version: "3.1.0",
		info: Info{
			Title:   "ohmyhttp API",
			Version: "0.1.0",
		},
		schemaName: defaultSchemaName,
	}
	for _, option := range options {
		option(&config)
	}
	return config
}

func cloneSecurity(requirements []SecurityRequirement) []SecurityRequirement {
	if len(requirements) == 0 {
		return nil
	}
	clone := make([]SecurityRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		item := make(SecurityRequirement, len(requirement))
		for name, scopes := range requirement {
			if scopes == nil {
				item[name] = []string{}
				continue
			}
			item[name] = append([]string(nil), scopes...)
		}
		clone = append(clone, item)
	}
	return clone
}
