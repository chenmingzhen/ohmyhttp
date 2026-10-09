package openapi

import (
	"fmt"
	"net/http"
	"reflect"
	"strconv"
	"strings"

	"github.com/chenmingzhen/ohmyhttp"
)

// EndpointSource 提供生成 OpenAPI 文档所需的强类型端点描述。
type EndpointSource interface {
	Endpoints() []ohmyhttp.EndpointInfo
}

// Build 从端点描述构建 OpenAPI 文档。
func Build(source EndpointSource, options ...Option) (*Spec, error) {
	config := newConfig(options)
	schemas := newSchemaBuilder(config)
	spec := &Spec{
		OpenAPI:  config.version,
		Info:     config.info,
		Servers:  config.servers,
		Paths:    map[string]*PathItem{},
		Security: cloneSecurity(config.security),
	}

	for _, route := range source.Endpoints() {
		operation, err := operationFromRoute(route, schemas)
		if err != nil {
			return nil, err
		}
		path := openAPIPath(route.Pattern)
		item := spec.Paths[path]
		if item == nil {
			item = &PathItem{}
			spec.Paths[path] = item
		}
		if err := assignOperation(item, route.Method, operation); err != nil {
			return nil, err
		}
	}

	if len(schemas.components) > 0 || len(config.securitySchemes) > 0 {
		spec.Components = &Components{}
		if len(schemas.components) > 0 {
			spec.Components.Schemas = schemas.components
		}
		if len(config.securitySchemes) > 0 {
			spec.Components.SecuritySchemes = config.securitySchemes
		}
	}

	normalizeSpec(spec, config.version)
	return spec, nil
}

var ohmyhttpResponseType = reflect.TypeOf(ohmyhttp.Response{})

func operationFromRoute(
	route ohmyhttp.EndpointInfo,
	schemas *schemaBuilder,
) (*Operation, error) {
	metadata := route.Metadata
	operation := &Operation{
		OperationID: metadata.Name,
		Summary:     metadata.Summary,
		Description: metadata.Description,
		Deprecated:  metadata.Deprecated,
		Tags:        append([]string(nil), metadata.Tags...),
		Responses:   responsesFromRoute(route, schemas),
		Security:    securityFromRoute(metadata.Security),
		Extensions:  extensionsFromValues(metadata.Values),
	}

	requestType := deref(route.RequestType)
	if requestType != nil {
		operation.Parameters = parametersFromRoute(requestType, schemas)
		if body := requestBodyFromRoute(route, requestType, schemas); body != nil {
			operation.RequestBody = body
		}
	}

	return operation, nil
}

func assignOperation(item *PathItem, method string, operation *Operation) error {
	switch strings.ToUpper(method) {
	case http.MethodGet:
		item.Get = operation
	case http.MethodPost:
		item.Post = operation
	case http.MethodPut:
		item.Put = operation
	case http.MethodPatch:
		item.Patch = operation
	case http.MethodDelete:
		item.Delete = operation
	case http.MethodHead:
		item.Head = operation
	case http.MethodOptions:
		item.Options = operation
	case http.MethodTrace:
		item.Trace = operation
	default:
		return fmt.Errorf("openapi: unsupported method %q", method)
	}
	return nil
}

func parametersFromRoute(requestType reflect.Type, schemas *schemaBuilder) []Parameter {
	fields := routeFields(requestType)
	parameters := make([]Parameter, 0)

	for _, field := range fields {
		for _, source := range []string{"path", "query", "header"} {
			name := tagName(field.StructField, source)
			if name == "" {
				continue
			}
			required := source == "path" || validateHas(field.StructField.Tag.Get("validate"), "required")
			schema := schemas.schemaFromType(field.Type)
			applyFieldTags(schema, field.StructField)
			parameters = append(parameters, Parameter{
				Name:     name,
				In:       source,
				Required: required,
				Schema:   schema,
			})
		}
	}

	return parameters
}

func requestBodyFromRoute(route ohmyhttp.EndpointInfo, requestType reflect.Type, schemas *schemaBuilder) *RequestBody {
	switch strings.ToUpper(route.Method) {
	case http.MethodPost, http.MethodPut, http.MethodPatch:
	default:
		return nil
	}

	content := map[string]MediaType{}
	if schema := schemas.taggedObjectSchema(requestType, "json"); schema != nil {
		content["application/json"] = MediaType{Schema: schema}
	}

	if schema := schemas.taggedObjectSchema(requestType, "form"); schema != nil {
		content["multipart/form-data"] = MediaType{Schema: schema}
		if !schemaHasBinary(schema) {
			content["application/x-www-form-urlencoded"] = MediaType{Schema: schema}
		}
	}

	if len(content) == 0 {
		return nil
	}
	return &RequestBody{
		Required: true,
		Content:  content,
	}
}

func responsesFromRoute(
	route ohmyhttp.EndpointInfo,
	schemas *schemaBuilder,
) map[string]Response {
	if len(route.Metadata.Returns) > 0 {
		responses := make(map[string]Response, len(route.Metadata.Returns))
		for _, item := range route.Metadata.Returns {
			responses[strconv.Itoa(item.Status)] = responseFromReturn(item, schemas)
		}
		return responses
	}

	status := route.SuccessStatus
	if status == 0 {
		status = http.StatusOK
	}
	item := ohmyhttp.Return{
		Status:      status,
		Description: statusDescription(status),
		BodyType:    route.ResponseType,
		ContentType: "application/json",
	}
	return map[string]Response{strconv.Itoa(status): responseFromReturn(item, schemas)}
}

func responseFromReturn(item ohmyhttp.Return, schemas *schemaBuilder) Response {
	response := Response{Description: item.Description}
	if response.Description == "" {
		response.Description = statusDescription(item.Status)
	}
	if statusDisallowsBody(item.Status) {
		return response
	}
	contentType := item.ContentType
	if contentType == "" {
		contentType = "application/json"
	}

	var schema *Schema
	switch {
	case item.Binary:
		schema = &Schema{Type: "string", Format: "binary"}
	case item.BodyType != nil && deref(item.BodyType) != ohmyhttpResponseType:
		schema = schemas.schemaFromType(item.BodyType)
	default:
		return response
	}
	response.Content = map[string]MediaType{
		contentType: {Schema: schema},
	}
	return response
}

func statusDescription(status int) string {
	if text := http.StatusText(status); text != "" {
		return text
	}
	return "response"
}

func statusDisallowsBody(status int) bool {
	return status >= 100 && status < 200 ||
		status == http.StatusNoContent ||
		status == http.StatusNotModified
}

func openAPIPath(pattern string) string {
	pattern = strings.TrimSuffix(pattern, "{$}")
	return strings.ReplaceAll(pattern, "...}", "}")
}

func extensionsFromValues(values map[string]any) map[string]any {
	extensions := map[string]any{}
	for key, value := range values {
		if strings.HasPrefix(key, "x-") {
			extensions[key] = value
		}
	}
	if len(extensions) == 0 {
		return nil
	}
	return extensions
}

func securityFromRoute(requirements []ohmyhttp.SecurityRequirement) []SecurityRequirement {
	if len(requirements) == 0 {
		return nil
	}
	security := make([]SecurityRequirement, 0, len(requirements))
	for _, requirement := range requirements {
		item := make(SecurityRequirement, len(requirement))
		for name, scopes := range requirement {
			if scopes == nil {
				item[name] = []string{}
				continue
			}
			item[name] = append([]string(nil), scopes...)
		}
		security = append(security, item)
	}
	return security
}
