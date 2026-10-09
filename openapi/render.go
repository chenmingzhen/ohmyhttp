package openapi

import (
	"encoding/json"

	"github.com/goccy/go-yaml"
)

// JSON 将 OpenAPI 文档编码为缩进后的 JSON。
func JSON(spec *Spec) ([]byte, error) {
	return json.MarshalIndent(spec, "", "  ")
}

// YAML 将 OpenAPI 文档编码为 YAML。
func YAML(spec *Spec) ([]byte, error) {
	return yaml.Marshal(spec)
}
