package openapi

import (
	"encoding/json"

	"github.com/goccy/go-yaml"
)

var (
	_ json.Marshaler          = Operation{}
	_ yaml.InterfaceMarshaler = Operation{}
	_ json.Marshaler          = SecurityScheme{}
	_ yaml.InterfaceMarshaler = SecurityScheme{}
)

func (operation Operation) MarshalJSON() ([]byte, error) {
	values := map[string]any{
		"responses": operation.Responses,
	}
	if operation.OperationID != "" {
		values["operationId"] = operation.OperationID
	}
	if operation.Summary != "" {
		values["summary"] = operation.Summary
	}
	if operation.Description != "" {
		values["description"] = operation.Description
	}
	if operation.Deprecated {
		values["deprecated"] = operation.Deprecated
	}
	if len(operation.Tags) > 0 {
		values["tags"] = operation.Tags
	}
	if len(operation.Parameters) > 0 {
		values["parameters"] = operation.Parameters
	}
	if operation.RequestBody != nil {
		values["requestBody"] = operation.RequestBody
	}
	if len(operation.Security) > 0 {
		values["security"] = operation.Security
	}
	addExtensions(values, operation.Extensions)
	return json.Marshal(values)
}

func (operation Operation) MarshalYAML() (any, error) {
	values := map[string]any{
		"responses": operation.Responses,
	}
	if operation.OperationID != "" {
		values["operationId"] = operation.OperationID
	}
	if operation.Summary != "" {
		values["summary"] = operation.Summary
	}
	if operation.Description != "" {
		values["description"] = operation.Description
	}
	if operation.Deprecated {
		values["deprecated"] = operation.Deprecated
	}
	if len(operation.Tags) > 0 {
		values["tags"] = operation.Tags
	}
	if len(operation.Parameters) > 0 {
		values["parameters"] = operation.Parameters
	}
	if operation.RequestBody != nil {
		values["requestBody"] = operation.RequestBody
	}
	if len(operation.Security) > 0 {
		values["security"] = operation.Security
	}
	addExtensions(values, operation.Extensions)
	return values, nil
}

func (scheme SecurityScheme) MarshalJSON() ([]byte, error) {
	values := map[string]any{"type": scheme.Type}
	if scheme.Description != "" {
		values["description"] = scheme.Description
	}
	if scheme.Name != "" {
		values["name"] = scheme.Name
	}
	if scheme.In != "" {
		values["in"] = scheme.In
	}
	if scheme.Scheme != "" {
		values["scheme"] = scheme.Scheme
	}
	if scheme.BearerFormat != "" {
		values["bearerFormat"] = scheme.BearerFormat
	}
	if scheme.OpenIDConnectURL != "" {
		values["openIdConnectUrl"] = scheme.OpenIDConnectURL
	}
	if scheme.Flows != nil {
		values["flows"] = scheme.Flows
	}
	addExtensions(values, scheme.Extensions)
	return json.Marshal(values)
}

func (scheme SecurityScheme) MarshalYAML() (any, error) {
	values := map[string]any{"type": scheme.Type}
	if scheme.Description != "" {
		values["description"] = scheme.Description
	}
	if scheme.Name != "" {
		values["name"] = scheme.Name
	}
	if scheme.In != "" {
		values["in"] = scheme.In
	}
	if scheme.Scheme != "" {
		values["scheme"] = scheme.Scheme
	}
	if scheme.BearerFormat != "" {
		values["bearerFormat"] = scheme.BearerFormat
	}
	if scheme.OpenIDConnectURL != "" {
		values["openIdConnectUrl"] = scheme.OpenIDConnectURL
	}
	if scheme.Flows != nil {
		values["flows"] = scheme.Flows
	}
	addExtensions(values, scheme.Extensions)
	return values, nil
}

func addExtensions(values map[string]any, extensions map[string]any) {
	for key, value := range extensions {
		if len(key) >= 2 && key[0] == 'x' && key[1] == '-' {
			values[key] = value
		}
	}
}
