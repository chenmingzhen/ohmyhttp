// Package errorx 在错误产生的位置保存调用堆栈，供调用方写日志。
//
// 这是一项可选能力。ohmyhttp 本体不引用本包，错误映射、恢复 panic、请求绑定和响应编码
// 都不依赖这里的类型。应用在自己的代码里 import 本包，在返回错误的位置调用 WithStack，
// 在错误观察器或日志里调用 StackTrace。
//
// 本包不定义 HTTP 状态码，也不定义 Problem。WithStack 不决定响应正文。
// StackError 实现了 Unwrap，映射器按内层错误继续匹配：内层是 HTTPError 时仍返回该业务错误，
// 内层是普通 error 时仍是 500 internal_error。响应正文不包含堆栈。
package errorx
