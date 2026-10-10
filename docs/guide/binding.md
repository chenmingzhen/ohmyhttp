# 请求绑定

端点的请求对象由框架创建。正文先解码，再写入查询参数、请求头和路径参数。校验在绑定之后、业务函数之前。

## JSON、路径和请求头

```go
type CreateReportRequest struct {
	TenantID      int    `path:"tenant_id" validate:"required,gt=0"`
	ClientVersion string `header:"X-Client-Version" validate:"required"`
	Name          string `json:"name" validate:"required"`
	Format        string `json:"format" validate:"required,oneof=csv pdf xlsx"`
}

type CreateReportResponse struct {
	ID            string `json:"id"`
	Name          string `json:"name"`
	Format        string `json:"format"`
	ClientVersion string `json:"client_version"`
}

func CreateReport(_ context.Context, request *CreateReportRequest) (CreateReportResponse, error) {
	return CreateReportResponse{
		ID:            "rpt_1",
		Name:          request.Name,
		Format:        request.Format,
		ClientVersion: request.ClientVersion,
	}, nil
}

app.Post("/tenants/{tenant_id}/reports", CreateReport)
```

请求：

```http
POST /tenants/7/reports HTTP/1.1
Content-Type: application/json
X-Client-Version: 1.0

{"name":"Monthly","format":"csv"}
```

响应：

```http
HTTP/1.1 200 OK
Content-Type: application/json; charset=utf-8

{"id":"rpt_1","name":"Monthly","format":"csv","client_version":"1.0"}
```

`Content-Type` 去掉参数后必须是 `application/json`。未知字段默认会绑定失败。`WithDiscardUnknownFields(true)` 改为忽略未知字段。空正文会跳过 JSON 解码，继续绑定其他来源。`{...}{...}` 这种多个顶层值会被拒绝。

正文里的 `"tenant_id": 1` 不会写进 `path:"tenant_id"`。带 `query`、`path`、`header`、`form` 的字段只从对应来源取值。

缺少 `Content-Type`，或媒体类型没有对应解码器时，响应是 415：

```json
{"code":"unsupported_media_type","message":"unsupported Content-Type \"text/plain\""}
```

## 查询参数

```go
type SearchRequest struct {
	Status string   `query:"status"`
	Tags   []string `query:"tag"`
	Page   int      `query:"page"`
}

type SearchResponse struct {
	Status string   `json:"status"`
	Tags   []string `json:"tags"`
	Page   int      `json:"page"`
}

func Search(_ context.Context, request *SearchRequest) (SearchResponse, error) {
	return SearchResponse{
		Status: request.Status,
		Tags:   request.Tags,
		Page:   request.Page,
	}, nil
}

app.Get("/reports", Search)
```

请求 `GET /reports?status=queued&tag=finance&tag=monthly&page=2` 的响应：

```json
{"status":"queued","tags":["finance","monthly"],"page":2}
```

切片接收重复参数。非切片字段收到重复参数时使用最后一个值。`*int` 收到 `"10"` 后指向 `10`。`bool` 接受 `true`、`false`、`1`、`0`。实现了 `encoding.TextUnmarshaler` 的类型用 `UnmarshalText` 解析。

## 表单

```go
type TokenRequest struct {
	GrantType string `form:"grant_type" validate:"required,oneof=password refresh_token"`
	Account   string `form:"account"`
	Password  string `form:"password"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
}

func IssueToken(_ context.Context, request *TokenRequest) (TokenResponse, error) {
	return TokenResponse{AccessToken: "tok_" + request.Account, TokenType: "Bearer"}, nil
}

app.Post("/oauth/token", IssueToken)
```

请求：

```http
POST /oauth/token HTTP/1.1
Content-Type: application/x-www-form-urlencoded

grant_type=password&account=demo&password=password
```

响应：

```json
{"access_token":"tok_demo","token_type":"Bearer"}
```

表单值来自 `PostForm`，查询参数不会混进来。

## 文件上传

文件字段使用 `*multipart.FileHeader`、`multipart.FileHeader` 或 `[]*multipart.FileHeader`。

```go
type UploadRequest struct {
	UserID      string                `path:"user_id" validate:"required"`
	DisplayName string                `form:"display_name" validate:"required"`
	Avatar      *multipart.FileHeader `form:"avatar" validate:"required"`
}

type UploadResponse struct {
	UserID   string `json:"user_id"`
	FileName string `json:"file_name"`
	Size     int64  `json:"size"`
}

func Upload(_ context.Context, request *UploadRequest) (UploadResponse, error) {
	return UploadResponse{
		UserID:   request.UserID,
		FileName: request.Avatar.Filename,
		Size:     request.Avatar.Size,
	}, nil
}

app.Post("/users/{user_id}/avatar", Upload, ohmyhttp.MaxBodyBytes(32<<20))
```

```bash
curl -F display_name=Ada -F avatar=@photo.png http://127.0.0.1:8080/users/42/avatar
```

```json
{"user_id":"42","file_name":"photo.png","size":2048}
```

小于 `WithMultipartMemory` 的部分留在内存，默认 8 MiB，超出部分写入临时文件。请求结束时临时文件会被删除。

## 来源标签

| 标签 | 取值位置 |
| --- | --- |
| `json` | `Content-Type: application/json` 的正文 |
| `form` | URL 编码表单或 multipart 的文本字段和文件 |
| `query` | URL 查询参数 |
| `header` | 请求头 |
| `path` | 路径参数。路由里没有这个片段时为空 |

一个字段只声明一种来源。`"-"` 表示忽略。匿名嵌入字段会展开，命名的嵌套结构体会继续向下绑定。
