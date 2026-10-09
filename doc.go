// Package ohmyhttp 提供直接构建在 net/http 上的强类型 HTTP 应用框架。
//
// App 管理路由和中间件。业务 Endpoint 接收 context.Context 和强类型请求；请求绑定、
// 校验、中间件、错误映射和响应编码均由 App 处理。App 本身实现 http.Handler，
// 并隐藏底层 ServeMux。
package ohmyhttp
