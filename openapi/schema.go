package openapi

import (
	"mime/multipart"
	"reflect"
	"strconv"
	"strings"
	"time"
)

// SchemaProvider 允许类型自定义自己的 OpenAPI schema。
type SchemaProvider interface {
	OpenAPISchema() *Schema
}

var (
	timeType      = reflect.TypeOf(time.Time{})
	fileHeader    = reflect.TypeOf(multipart.FileHeader{})
	fileHeaderPtr = reflect.TypeOf((*multipart.FileHeader)(nil))
)

type routeField struct {
	reflect.StructField
	Type reflect.Type
}

type schemaBuilder struct {
	config     config
	components map[string]*Schema
	building   map[reflect.Type]bool
}

func newSchemaBuilder(config config) *schemaBuilder {
	return &schemaBuilder{
		config:     config,
		components: map[string]*Schema{},
		building:   map[reflect.Type]bool{},
	}
}

func (builder *schemaBuilder) schemaFromType(t reflect.Type) *Schema {
	nullable := false
	for t != nil && t.Kind() == reflect.Pointer {
		nullable = true
		t = t.Elem()
	}
	schema := builder.schemaFromConcreteType(t)
	if nullable && schema != nil {
		schema = cloneSchema(schema)
		schema.Nullable = true
	}
	return schema
}

func (builder *schemaBuilder) schemaFromConcreteType(t reflect.Type) *Schema {
	if t == nil {
		return &Schema{}
	}
	if custom := schemaFromProvider(t); custom != nil {
		return builder.customize(t, custom)
	}
	if t == timeType {
		return builder.customize(t, &Schema{Type: "string", Format: "date-time"})
	}
	if t == fileHeader || t == fileHeaderPtr {
		return builder.customize(t, &Schema{Type: "string", Format: "binary"})
	}

	switch t.Kind() {
	case reflect.Bool:
		return builder.customize(t, &Schema{Type: "boolean"})
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return builder.customize(t, &Schema{Type: "integer"})
	case reflect.Float32, reflect.Float64:
		return builder.customize(t, &Schema{Type: "number"})
	case reflect.String:
		return builder.customize(t, &Schema{Type: "string"})
	case reflect.Slice, reflect.Array:
		if t.Elem().Kind() == reflect.Uint8 {
			return builder.customize(t, &Schema{Type: "string", Format: "binary"})
		}
		return builder.customize(t, &Schema{Type: "array", Items: builder.schemaFromType(t.Elem())})
	case reflect.Map:
		return builder.customize(t, &Schema{
			Type:                 "object",
			AdditionalProperties: builder.schemaFromType(t.Elem()),
		})
	case reflect.Struct:
		if name := builder.componentName(t); name != "" {
			builder.ensureComponent(name, t)
			return &Schema{Ref: "#/components/schemas/" + name}
		}
		return builder.customize(t, builder.objectSchema(t, ""))
	default:
		return builder.customize(t, &Schema{})
	}
}

func (builder *schemaBuilder) ensureComponent(name string, t reflect.Type) {
	if _, ok := builder.components[name]; ok {
		return
	}
	builder.components[name] = &Schema{Type: "object"}
	if builder.building[t] {
		return
	}
	builder.building[t] = true
	builder.components[name] = builder.customize(t, builder.objectSchema(t, ""))
	delete(builder.building, t)
}

func (builder *schemaBuilder) componentName(t reflect.Type) string {
	if builder.config.schemaName == nil {
		return defaultSchemaName(t)
	}
	return strings.TrimSpace(builder.config.schemaName(t))
}

func defaultSchemaName(t reflect.Type) string {
	t = deref(t)
	if t == nil || t.Kind() != reflect.Struct || t.Name() == "" {
		return ""
	}
	if t.PkgPath() == "" {
		return t.Name()
	}
	parts := strings.Split(t.PkgPath(), "/")
	pkg := ""
	if len(parts) > 0 {
		pkg = parts[len(parts)-1]
	}
	pkg = strings.NewReplacer("-", "_", ".", "_").Replace(pkg)
	if pkg == "" {
		return t.Name()
	}
	return exportSchemaName(pkg) + t.Name()
}

func exportSchemaName(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToUpper(value[:1]) + value[1:]
}

func (builder *schemaBuilder) customize(t reflect.Type, schema *Schema) *Schema {
	if schema == nil {
		return &Schema{}
	}
	if builder.config.schemaCustomize == nil {
		return schema
	}
	if customized := builder.config.schemaCustomize(t, cloneSchema(schema)); customized != nil {
		return customized
	}
	return schema
}

func schemaFromProvider(t reflect.Type) *Schema {
	if t == nil {
		return nil
	}
	providerType := reflect.TypeOf((*SchemaProvider)(nil)).Elem()
	if t.Kind() != reflect.Pointer && reflect.PointerTo(t).Implements(providerType) {
		value := reflect.New(t).Interface().(SchemaProvider)
		return value.OpenAPISchema()
	}
	if t.Implements(providerType) {
		value := reflect.New(t).Elem()
		if value.CanInterface() {
			return value.Interface().(SchemaProvider).OpenAPISchema()
		}
	}
	return nil
}

func (builder *schemaBuilder) objectSchema(t reflect.Type, tag string) *Schema {
	schema := &Schema{Type: "object", Properties: map[string]*Schema{}}
	for _, field := range routeFields(t) {
		name := fieldName(field.StructField, tag)
		if name == "" {
			continue
		}
		property := builder.schemaFromType(field.Type)
		applyFieldTags(property, field.StructField)
		schema.Properties[name] = property
		if validateHas(field.StructField.Tag.Get("validate"), "required") {
			schema.Required = append(schema.Required, name)
		}
	}
	if len(schema.Properties) == 0 {
		return &Schema{Type: "object"}
	}
	return schema
}

func (builder *schemaBuilder) taggedObjectSchema(t reflect.Type, tag string) *Schema {
	schema := &Schema{Type: "object", Properties: map[string]*Schema{}}
	for _, field := range routeFields(t) {
		name := tagName(field.StructField, tag)
		if name == "" {
			continue
		}
		property := builder.schemaFromType(field.Type)
		applyFieldTags(property, field.StructField)
		schema.Properties[name] = property
		if validateHas(field.StructField.Tag.Get("validate"), "required") {
			schema.Required = append(schema.Required, name)
		}
	}
	if len(schema.Properties) == 0 {
		return nil
	}
	return schema
}

func schemaHasBinary(schema *Schema) bool {
	if schema == nil {
		return false
	}
	if schema.Type == "string" && schema.Format == "binary" {
		return true
	}
	if schema.Items != nil && schemaHasBinary(schema.Items) {
		return true
	}
	for _, property := range schema.Properties {
		if schemaHasBinary(property) {
			return true
		}
	}
	if additional, ok := schema.AdditionalProperties.(*Schema); ok {
		return schemaHasBinary(additional)
	}
	return false
}

func routeFields(t reflect.Type) []routeField {
	t = deref(t)
	if t == nil || t.Kind() != reflect.Struct {
		return nil
	}
	fields := make([]routeField, 0, t.NumField())
	for index := 0; index < t.NumField(); index++ {
		field := t.Field(index)
		if field.PkgPath != "" {
			continue
		}
		fieldType := deref(field.Type)
		if field.Anonymous && fieldType != nil && fieldType.Kind() == reflect.Struct {
			fields = append(fields, routeFields(field.Type)...)
			continue
		}
		fields = append(fields, routeField{StructField: field, Type: field.Type})
	}
	return fields
}

func fieldName(field reflect.StructField, preferredTag string) string {
	if preferredTag != "" {
		if tagIgnored(field, preferredTag) {
			return ""
		}
		if name := tagName(field, preferredTag); name != "" {
			return name
		}
	}
	for _, tag := range []string{"json", "form"} {
		if tagIgnored(field, tag) {
			return ""
		}
		if name := tagName(field, tag); name != "" {
			return name
		}
	}
	return lowerCamel(field.Name)
}

func tagIgnored(field reflect.StructField, tag string) bool {
	return strings.SplitN(field.Tag.Get(tag), ",", 2)[0] == "-"
}

func tagName(field reflect.StructField, tag string) string {
	value := strings.SplitN(field.Tag.Get(tag), ",", 2)[0]
	if value == "-" {
		return ""
	}
	return strings.TrimSpace(value)
}

func applyFieldTags(schema *Schema, field reflect.StructField) {
	if schema == nil {
		return
	}
	applyValidateTag(schema, field.Tag.Get("validate"))
	if description := firstTag(field, "description", "doc"); description != "" {
		schema.Description = description
	}
	if format := strings.TrimSpace(field.Tag.Get("format")); format != "" {
		schema.Format = format
	}
	if pattern := strings.TrimSpace(field.Tag.Get("pattern")); pattern != "" {
		schema.Pattern = pattern
	}
	if value := strings.TrimSpace(field.Tag.Get("default")); value != "" {
		schema.Default = parseLiteral(schema, value)
	}
	if value := strings.TrimSpace(field.Tag.Get("example")); value != "" {
		schema.Example = parseLiteral(schema, value)
	}
	if value := strings.TrimSpace(field.Tag.Get("enum")); value != "" {
		for _, item := range splitList(value) {
			schema.Enum = append(schema.Enum, parseLiteral(schema, item))
		}
	}
	if tagBool(field.Tag.Get("nullable")) || isNullableField(field.Type) {
		schema.Nullable = true
	}
	if tagBool(field.Tag.Get("readOnly")) {
		schema.ReadOnly = true
	}
	if tagBool(field.Tag.Get("writeOnly")) {
		schema.WriteOnly = true
	}
}

func firstTag(field reflect.StructField, names ...string) string {
	for _, name := range names {
		if value := strings.TrimSpace(field.Tag.Get(name)); value != "" {
			return value
		}
	}
	return ""
}

func applyValidateTag(schema *Schema, tag string) {
	for _, rule := range strings.Split(tag, ",") {
		name, parameter, _ := strings.Cut(rule, "=")
		switch name {
		case "email":
			schema.Format = "email"
		case "url", "uri":
			schema.Format = "uri"
		case "uuid", "uuid4", "uuid5":
			schema.Format = "uuid"
		case "datetime":
			schema.Format = "date-time"
		case "hostname":
			schema.Format = "hostname"
		case "ip", "ipv4", "ipv6":
			schema.Format = name
		case "oneof":
			for _, value := range strings.Fields(parameter) {
				schema.Enum = append(schema.Enum, parseLiteral(schema, value))
			}
		case "min", "gte":
			applyMinimum(schema, parameter)
		case "max", "lte":
			applyMaximum(schema, parameter)
		case "gt":
			applyExclusiveMinimum(schema, parameter)
		case "lt":
			applyExclusiveMaximum(schema, parameter)
		case "len":
			applyLength(schema, parameter)
		}
	}
}

func applyMinimum(schema *Schema, parameter string) {
	if schema.Type == "string" {
		if value, err := strconv.Atoi(parameter); err == nil {
			schema.MinLength = &value
		}
		return
	}
	if schema.Type == "array" {
		if value, err := strconv.Atoi(parameter); err == nil {
			schema.MinItems = &value
		}
		return
	}
	if value, err := strconv.ParseFloat(parameter, 64); err == nil {
		schema.Minimum = &value
	}
}

func applyMaximum(schema *Schema, parameter string) {
	if schema.Type == "string" {
		if value, err := strconv.Atoi(parameter); err == nil {
			schema.MaxLength = &value
		}
		return
	}
	if schema.Type == "array" {
		if value, err := strconv.Atoi(parameter); err == nil {
			schema.MaxItems = &value
		}
		return
	}
	if value, err := strconv.ParseFloat(parameter, 64); err == nil {
		schema.Maximum = &value
	}
}

func applyExclusiveMinimum(schema *Schema, parameter string) {
	if value, err := strconv.ParseFloat(parameter, 64); err == nil {
		schema.ExclusiveMinimum = value
	}
}

func applyExclusiveMaximum(schema *Schema, parameter string) {
	if value, err := strconv.ParseFloat(parameter, 64); err == nil {
		schema.ExclusiveMaximum = value
	}
}

func applyLength(schema *Schema, parameter string) {
	value, err := strconv.Atoi(parameter)
	if err != nil {
		return
	}
	switch schema.Type {
	case "string":
		schema.MinLength = &value
		schema.MaxLength = &value
	case "array":
		schema.MinItems = &value
		schema.MaxItems = &value
	}
}

func validateHas(tag string, expected string) bool {
	for _, rule := range strings.Split(tag, ",") {
		name, _, _ := strings.Cut(rule, "=")
		if name == expected {
			return true
		}
	}
	return false
}

func deref(t reflect.Type) reflect.Type {
	for t != nil && t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	return t
}

func isNullableField(t reflect.Type) bool {
	return t != nil && t.Kind() == reflect.Pointer
}

func tagBool(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "true", "1", "yes":
		return true
	default:
		return false
	}
}

func parseLiteral(schema *Schema, value string) any {
	switch schema.Type {
	case "boolean":
		if parsed, err := strconv.ParseBool(value); err == nil {
			return parsed
		}
	case "integer":
		if parsed, err := strconv.ParseInt(value, 10, 64); err == nil {
			return parsed
		}
	case "number":
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return value
}

func splitList(value string) []string {
	if strings.Contains(value, ",") {
		parts := strings.Split(value, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				values = append(values, part)
			}
		}
		return values
	}
	return strings.Fields(value)
}

func lowerCamel(value string) string {
	if value == "" {
		return ""
	}
	return strings.ToLower(value[:1]) + value[1:]
}

func cloneSchema(schema *Schema) *Schema {
	if schema == nil {
		return nil
	}
	clone := *schema
	if len(schema.Properties) > 0 {
		clone.Properties = make(map[string]*Schema, len(schema.Properties))
		for name, property := range schema.Properties {
			clone.Properties[name] = cloneSchema(property)
		}
	}
	clone.Items = cloneSchema(schema.Items)
	if additional, ok := schema.AdditionalProperties.(*Schema); ok {
		clone.AdditionalProperties = cloneSchema(additional)
	}
	clone.Required = append([]string(nil), schema.Required...)
	clone.Enum = append([]any(nil), schema.Enum...)
	return &clone
}

func schemaFromType(t reflect.Type) *Schema {
	return newSchemaBuilder(newConfig(nil)).schemaFromType(t)
}

func objectSchema(t reflect.Type, tag string) *Schema {
	return newSchemaBuilder(newConfig(nil)).objectSchema(t, tag)
}

func taggedObjectSchema(t reflect.Type, tag string) *Schema {
	return newSchemaBuilder(newConfig(nil)).taggedObjectSchema(t, tag)
}
