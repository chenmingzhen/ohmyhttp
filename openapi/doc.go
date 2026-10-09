// Package openapi 根据 ohmyhttp.App 的路由快照构建 OpenAPI 3.1 或 3.0.3 文档。
//
// 这个包只读取 ohmyhttp 的强类型路由描述，不访问内部 net/http 路由树。
// 因此它描述的是应用层契约，而不是底层框架的全部传输行为。
//
// JSONHandler、YAMLHandler、SchemasHandler 和 DocsHandler 返回 ohmyhttp.ResponseFunc，
// 可通过 App.HandleResponse 挂载，用于在运行时暴露 /openapi.json、
// /openapi.yaml、/schemas 或 /docs。
package openapi
