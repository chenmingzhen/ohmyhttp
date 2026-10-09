// Package meta 提供路由描述信息选项。
//
// 这些选项不会改变请求处理流程，只用于日志、指标、OpenAPI 和代码生成等场景。
package meta

import (
	"reflect"
	"strings"

	"github.com/chenmingzhen/ohmyhttp"
)

// Name 设置路由的稳定名称，通常对应 operationId。
func Name(name string) ohmyhttp.MetadataOption {
	return func(metadata *ohmyhttp.RouteMetadata) {
		metadata.Name = strings.TrimSpace(name)
	}
}

// Summary 设置路由的短摘要。
func Summary(summary string) ohmyhttp.MetadataOption {
	return func(metadata *ohmyhttp.RouteMetadata) {
		metadata.Summary = strings.TrimSpace(summary)
	}
}

// Description 设置路由的详细描述。
func Description(description string) ohmyhttp.MetadataOption {
	return func(metadata *ohmyhttp.RouteMetadata) {
		metadata.Description = strings.TrimSpace(description)
	}
}

// Deprecated 标记路由已经废弃。
func Deprecated() ohmyhttp.MetadataOption {
	return func(metadata *ohmyhttp.RouteMetadata) {
		metadata.Deprecated = true
	}
}

// Tags 设置路由标签，空标签会被忽略。
func Tags(tags ...string) ohmyhttp.MetadataOption {
	return func(metadata *ohmyhttp.RouteMetadata) {
		metadata.Tags = cleanTags(tags)
	}
}

// Value 设置自定义元数据。它适合放业务无关的补充描述，例如审计标记、权限标签等。
func Value(key string, value any) ohmyhttp.MetadataOption {
	return func(metadata *ohmyhttp.RouteMetadata) {
		key = strings.TrimSpace(key)
		if key == "" {
			panic("ohmyhttp/meta: route metadata key is empty")
		}
		if metadata.Values == nil {
			metadata.Values = make(map[string]any)
		}
		metadata.Values[key] = value
	}
}

// Extension 设置 OpenAPI 扩展字段。key 必须以 x- 开头。
func Extension(key string, value any) ohmyhttp.MetadataOption {
	return func(metadata *ohmyhttp.RouteMetadata) {
		key = strings.TrimSpace(key)
		if !strings.HasPrefix(key, "x-") {
			panic("ohmyhttp/meta: OpenAPI extension key must start with x-")
		}
		if metadata.Values == nil {
			metadata.Values = make(map[string]any)
		}
		metadata.Values[key] = value
	}
}

// Security 声明当前路由的 OpenAPI security requirement。
//
// 它只影响文档，不会自动添加鉴权中间件。
func Security(requirements ...ohmyhttp.SecurityRequirement) ohmyhttp.MetadataOption {
	return func(metadata *ohmyhttp.RouteMetadata) {
		metadata.Security = cloneSecurity(requirements)
	}
}

// Returns 声明一个 JSON 响应。
func Returns(status int, description string, body any) ohmyhttp.MetadataOption {
	return returns(status, description, "application/json", body)
}

// ReturnsError 声明一个默认 application/json 错误响应。
func ReturnsError(status int, description string) ohmyhttp.MetadataOption {
	return Returns(status, description, (*ohmyhttp.Problem)(nil))
}

// ReturnsContent 声明一个指定 Content-Type 的响应。
func ReturnsContent(status int, description string, contentType string, body any) ohmyhttp.MetadataOption {
	return returns(status, description, contentType, body)
}

// ReturnsFile 声明一个文件下载响应。
func ReturnsFile(status int, contentType string, description string) ohmyhttp.MetadataOption {
	return binaryReturn(status, contentType, description)
}

// ReturnsStream 声明一个流式响应。
func ReturnsStream(status int, contentType string, description string) ohmyhttp.MetadataOption {
	return binaryReturn(status, contentType, description)
}

func returns(status int, description string, contentType string, body any) ohmyhttp.MetadataOption {
	return func(metadata *ohmyhttp.RouteMetadata) {
		metadata.Returns = append(metadata.Returns, ohmyhttp.Return{
			Status:      status,
			Description: strings.TrimSpace(description),
			BodyType:    typeFromValue(body),
			ContentType: strings.TrimSpace(contentType),
		})
	}
}

func binaryReturn(status int, contentType string, description string) ohmyhttp.MetadataOption {
	return func(metadata *ohmyhttp.RouteMetadata) {
		metadata.Returns = append(metadata.Returns, ohmyhttp.Return{
			Status:      status,
			Description: strings.TrimSpace(description),
			ContentType: strings.TrimSpace(contentType),
			Binary:      true,
		})
	}
}

func cleanTags(tags []string) []string {
	cleaned := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag != "" {
			cleaned = append(cleaned, tag)
		}
	}
	return cleaned
}

func typeFromValue(value any) reflect.Type {
	if value == nil {
		return nil
	}
	return reflect.TypeOf(value)
}

func cloneSecurity(requirements []ohmyhttp.SecurityRequirement) []ohmyhttp.SecurityRequirement {
	if len(requirements) == 0 {
		return nil
	}
	clone := make([]ohmyhttp.SecurityRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		item := make(ohmyhttp.SecurityRequirement, len(requirement))
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
