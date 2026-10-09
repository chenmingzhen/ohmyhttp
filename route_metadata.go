package ohmyhttp

import (
	"maps"
	"reflect"
	"slices"
)

// RouteMetadata 是不影响运行时匹配的路由元数据，可用于日志、指标和 OpenAPI。
type RouteMetadata struct {
	Name        string
	Summary     string
	Description string
	Deprecated  bool
	Tags        []string
	Returns     []Return
	Security    []SecurityRequirement
	Values      map[string]any
}

// SecurityRequirement 描述一个 OpenAPI security requirement。它只用于路由元数据，
// 不会改变运行时鉴权逻辑。
type SecurityRequirement map[string][]string

// Return 描述路由可能返回的 HTTP 响应，供 OpenAPI 使用。
type Return struct {
	Status      int
	Description string
	BodyType    reflect.Type
	ContentType string
	Binary      bool
}

// RouteOption 可同时用于强类型端点路由和 ResponseFunc 路由。
type RouteOption interface {
	EndpointOption
	ResponseRouteOption
}

// MetadataOption 在注册阶段定制路由描述信息。
type MetadataOption func(*RouteMetadata)

// Describe 把路由描述信息集中到一个选项中，避免和运行时行为选项混在一起。
func Describe(options ...MetadataOption) RouteOption {
	return describeRouteOption{options: options}
}

type describeRouteOption struct {
	options []MetadataOption
}

var _ RouteOption = describeRouteOption{}

func (option describeRouteOption) applyEndpoint(route *endpointRoute) {
	option.apply(&route.metadata)
}

func (option describeRouteOption) applyResponseRoute(route *responseRoute) {
	option.apply(&route.metadata)
}

func (option describeRouteOption) apply(metadata *RouteMetadata) {
	for _, apply := range option.options {
		apply(metadata)
	}
}

func cloneRouteMetadata(metadata RouteMetadata) RouteMetadata {
	cloned := metadata
	cloned.Tags = slices.Clone(metadata.Tags)
	cloned.Returns = slices.Clone(metadata.Returns)
	cloned.Values = maps.Clone(metadata.Values)
	if metadata.Security != nil {
		cloned.Security = make([]SecurityRequirement, len(metadata.Security))
		for index, requirement := range metadata.Security {
			clonedRequirement := make(SecurityRequirement, len(requirement))
			for name, scopes := range requirement {
				clonedRequirement[name] = slices.Clone(scopes)
			}
			cloned.Security[index] = clonedRequirement
		}
	}
	return cloned
}
