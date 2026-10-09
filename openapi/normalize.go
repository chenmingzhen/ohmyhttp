package openapi

import "strings"

func normalizeSpec(spec *Spec, version string) {
	if spec == nil {
		return
	}
	if strings.HasPrefix(version, "3.0") {
		normalizeSpecForOpenAPI30(spec)
	}
}

func normalizeSpecForOpenAPI30(spec *Spec) {
	if spec.Components != nil {
		for _, schema := range spec.Components.Schemas {
			normalizeSchemaForOpenAPI30(schema)
		}
	}
	for _, path := range spec.Paths {
		for _, operation := range operations(path) {
			if operation == nil {
				continue
			}
			for index := range operation.Parameters {
				normalizeSchemaForOpenAPI30(operation.Parameters[index].Schema)
			}
			if operation.RequestBody != nil {
				for _, mediaType := range operation.RequestBody.Content {
					normalizeSchemaForOpenAPI30(mediaType.Schema)
				}
			}
			for _, response := range operation.Responses {
				for _, header := range response.Headers {
					normalizeSchemaForOpenAPI30(header.Schema)
				}
				for _, mediaType := range response.Content {
					normalizeSchemaForOpenAPI30(mediaType.Schema)
				}
			}
		}
	}
}

func operations(path *PathItem) []*Operation {
	if path == nil {
		return nil
	}
	return []*Operation{
		path.Get,
		path.Post,
		path.Put,
		path.Patch,
		path.Delete,
		path.Head,
		path.Options,
		path.Trace,
	}
}

func normalizeSchemaForOpenAPI30(schema *Schema) {
	if schema == nil {
		return
	}
	if minimum, ok := exclusiveNumber(schema.ExclusiveMinimum); ok {
		schema.Minimum = &minimum
		schema.ExclusiveMinimum = true
	}
	if maximum, ok := exclusiveNumber(schema.ExclusiveMaximum); ok {
		schema.Maximum = &maximum
		schema.ExclusiveMaximum = true
	}
	for _, property := range schema.Properties {
		normalizeSchemaForOpenAPI30(property)
	}
	normalizeSchemaForOpenAPI30(schema.Items)
	if additional, ok := schema.AdditionalProperties.(*Schema); ok {
		normalizeSchemaForOpenAPI30(additional)
	}
}

func exclusiveNumber(value any) (float64, bool) {
	switch typed := value.(type) {
	case float64:
		return typed, true
	case *float64:
		if typed != nil {
			return *typed, true
		}
	}
	return 0, false
}
