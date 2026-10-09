package demo

import (
	"reflect"

	"github.com/chenmingzhen/ohmyhttp/openapi"
)

// OpenAPITitle 是示例 API 文档标题。
const OpenAPITitle = "ohmyhttp demo API"

// OpenAPIOptions 返回示例 App 生成 OpenAPI 时使用的公共配置。
func OpenAPIOptions() []openapi.Option {
	return []openapi.Option{
		openapi.WithOpenAPIVersion("3.1.0"),
		openapi.WithInfo(OpenAPITitle, "0.1.0"),
		openapi.WithDescription("由 ohmyhttp 强类型路由元数据生成的 OpenAPI 文档。"),
		openapi.WithServer("http://127.0.0.1:8080", "本地示例服务"),
		openapi.WithSecurityScheme("bearerAuth", openapi.SecurityScheme{
			Type:         "http",
			Scheme:       "bearer",
			BearerFormat: "JWT",
			Description:  "通过登录接口签发的示例 JWT。",
		}),
		openapi.WithSchemaCustomizer(func(t reflect.Type, schema *openapi.Schema) *openapi.Schema {
			if t == reflect.TypeOf(ReportID("")) {
				schema.Description = "报表 ID。示例值包括 monthly、secret、missing。"
				schema.Pattern = "^[a-z][a-z0-9-]*$"
				schema.Example = "monthly"
			}
			return schema
		}),
	}
}
