package openapi

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/chenmingzhen/ohmyhttp"
)

type Format string

const (
	FormatJSON Format = "json"
	FormatYAML Format = "yaml"
)

// SpecHandler 返回由 ohmyhttp 统一编码和观察的 OpenAPI 响应函数。
func SpecHandler(
	source EndpointSource,
	format Format,
	options ...Option,
) ohmyhttp.ResponseFunc {
	return func(_ *http.Request) (ohmyhttp.Response, error) {
		spec, err := Build(source, options...)
		if err != nil {
			return ohmyhttp.Response{}, err
		}
		data, contentType, err := renderSpec(spec, format)
		if err != nil {
			return ohmyhttp.Response{}, err
		}
		return ohmyhttp.Bytes(http.StatusOK, contentType, append(data, '\n')), nil
	}
}

func JSONHandler(source EndpointSource, options ...Option) ohmyhttp.ResponseFunc {
	return SpecHandler(source, FormatJSON, options...)
}

func YAMLHandler(source EndpointSource, options ...Option) ohmyhttp.ResponseFunc {
	return SpecHandler(source, FormatYAML, options...)
}

func SchemasHandler(source EndpointSource, options ...Option) ohmyhttp.ResponseFunc {
	return func(request *http.Request) (ohmyhttp.Response, error) {
		spec, err := Build(source, options...)
		if err != nil {
			return ohmyhttp.Response{}, err
		}
		schemas := map[string]*Schema{}
		if spec.Components != nil && spec.Components.Schemas != nil {
			schemas = spec.Components.Schemas
		}

		var value any = schemas
		if name := request.URL.Query().Get("name"); name != "" {
			schema, ok := schemas[name]
			if !ok {
				return ohmyhttp.Response{}, ohmyhttp.NewError(
					http.StatusNotFound,
					"schema_not_found",
					"schema does not exist",
				)
			}
			value = schema
		}
		data, err := json.MarshalIndent(value, "", "  ")
		if err != nil {
			return ohmyhttp.Response{}, err
		}
		return ohmyhttp.Bytes(
			http.StatusOK,
			"application/json; charset=utf-8",
			append(data, '\n'),
		), nil
	}
}

func renderSpec(spec *Spec, format Format) ([]byte, string, error) {
	switch format {
	case FormatJSON:
		data, err := JSON(spec)
		return data, "application/json; charset=utf-8", err
	case FormatYAML:
		data, err := YAML(spec)
		return data, "application/yaml; charset=utf-8", err
	default:
		return nil, "", fmt.Errorf("openapi: unsupported format %q", format)
	}
}
