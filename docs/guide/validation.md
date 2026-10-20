# 请求校验

绑定成功后，框架按 `validate` 标签检查字段。标签失败就返回，不再做跨字段校验。标签来自 [go-playground/validator](https://github.com/go-playground/validator)。

## 字段规则

```go
type UpdateProfileRequest struct {
	UserID  string             `header:"X-User-ID" validate:"required"`
	Profile *UpdateProfileBody `json:"profile" validate:"required"`
}

type UpdateProfileBody struct {
	DisplayName string  `json:"display_name" validate:"required,min=2,max=40"`
	Bio         *string `json:"bio,omitempty" validate:"omitempty,max=120"`
}

type UpdateProfileResponse struct {
	UserID      string `json:"user_id"`
	DisplayName string `json:"display_name"`
}

func UpdateProfile(_ context.Context, request *UpdateProfileRequest) (UpdateProfileResponse, error) {
	return UpdateProfileResponse{
		UserID:      request.UserID,
		DisplayName: request.Profile.DisplayName,
	}, nil
}

app.Patch("/profile", UpdateProfile)
```

请求：

```http
PATCH /profile HTTP/1.1
Content-Type: application/json
X-User-ID: user-1

{"profile":{"display_name":"A"}}
```

`display_name` 少于 2 个字符，响应是 400：

```json
{
  "code": "validation_failed",
  "message": "request validation failed",
  "details": [
    { "field": "profile.display_name", "rule": "min", "parameter": "2" }
  ]
}
```

字段名按 `json`、`query`、`path`、`header`、`form` 的顺序取第一个可用标签。`json:"display_name,omitempty"` 只保留逗号前的名字。嵌套字段去掉根结构体名，所以上面是 `profile.display_name`。

同一请求把 `display_name` 改成 `"Ada"` 时，响应是：

```json
{ "user_id": "user-1", "display_name": "Ada" }
```

`details` 里每一项是 `Violation`：`field` 是字段路径，`rule` 是规则名，`parameter` 是规则参数，`message` 可由 `NewValidationError` 自行填写。

## 跨字段校验

标签全部通过后，如果请求实现了 `RequestValidator`，再调用 `ValidateRequest`。

```go
type CreateUserRequest struct {
	Password        string `json:"password" validate:"required,min=8"`
	ConfirmPassword string `json:"confirm_password" validate:"required"`
}

func (r *CreateUserRequest) ValidateRequest(context.Context) error {
	if r.Password != r.ConfirmPassword {
		return ohmyhttp.NewValidationError(
			fmt.Errorf("password and confirm_password do not match"),
			[]ohmyhttp.Violation{{
				Field:   "confirm_password",
				Rule:    "password_match",
				Message: "password and confirm_password must match",
			}},
		)
	}
	return nil
}
```

`{"password":"password","confirm_password":"different"}` 的响应：

```json
{
  "code": "validation_failed",
  "message": "request validation failed",
  "details": [
    {
      "field": "confirm_password",
      "rule": "password_match",
      "message": "password and confirm_password must match"
    }
  ]
}
```

`NewValidationError` 的 `Details` 原样进入响应。返回普通 `error` 时没有 `details`。`ValidateRequest` 可以读取 `context.Context`。例如先用 `oneof` 限制 `grant_type`，再在这里检查该模式下对应字段是否都有值。

希望校验失败使用 422 时，用 `WithErrorMapper` 识别 `*ValidationError`，见 [错误处理](/guide/errors)。

## 自定义规则

```go
app := ohmyhttp.New(
	ohmyhttp.WithValidationRule("username", func(field playground.FieldLevel) bool {
		value := field.Field().String()
		if len(value) < 3 || len(value) > 32 || !isLetter(rune(value[0])) {
			return false
		}
		for _, item := range value {
			if !isLetter(item) && !isDigit(item) && item != '_' {
				return false
			}
		}
		return true
	}),
)

func isLetter(value rune) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func isDigit(value rune) bool {
	return value >= '0' && value <= '9'
}
```

字段写成 `validate:"required,username"`。规则返回 true 表示通过。第三个参数为 true 时，字段值是 nil 也会调用规则；默认跳过 nil。

```go
type CreateUserRequest struct {
	Username string `json:"username" validate:"required,username"`
}
```

`{"username":"1bad"}` 会返回 400，`rule` 为 `username`。`{"username":"ada"}` 通过这条规则。

同一套 `validate` 标签会进入 OpenAPI Schema。见 [OpenAPI](/guide/openapi#schema)。

## protobuf 校验

protobuf 请求不使用 Go struct 的 `validate` 标签，而是通过
`WithProtoValidator` 注入校验器。校验规则通常写在 `.proto` 中。
