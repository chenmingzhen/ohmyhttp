package ohmyhttp

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"

	playground "github.com/go-playground/validator/v10"
	"google.golang.org/protobuf/proto"
)

// Violation 是适合通过 HTTP 传输的字段校验详情。
type Violation struct {
	Field     string `json:"field"`
	Rule      string `json:"rule"`
	Parameter string `json:"parameter,omitempty"`
	Message   string `json:"message,omitempty"`
}

// RequestValidator 由需要自定义请求级校验的请求结构实现，适合跨字段规则或需要
// context.Context 的校验。
type RequestValidator interface {
	ValidateRequest(context.Context) error
}

// ProtoValidator 是 protobuf 请求的可选校验器。框架不会默认引入某一种
// protobuf 校验实现；业务方可以注入 protovalidate、protoc-gen-validate
// 或自己的校验逻辑。
type ProtoValidator func(context.Context, proto.Message) error

// WithValidationRule 注册一个 go-playground/validator 字段级校验规则。
// tag 写在请求字段的 validate 标签里，例如 validate:"required,username"。
// rule 返回 true 表示这个字段通过。callValidationEvenIfNull 为 true 时，
// 字段值是 nil 也会调用 rule；默认跳过 nil 字段。
// 注册发生在构建 App 时，标签为空或注册失败会直接 panic。
func WithValidationRule(
	tag string,
	rule playground.Func,
	callValidationEvenIfNull ...bool,
) AppOption {
	return func(runtime *appRuntime) {
		tag = strings.TrimSpace(tag)
		if tag == "" {
			panic("ohmyhttp: validation rule tag is empty")
		}
		if err := runtime.validator.structs.RegisterValidation(
			tag,
			rule,
			callValidationEvenIfNull...,
		); err != nil {
			panic(fmt.Sprintf("ohmyhttp: register validation rule %q: %v", tag, err))
		}
	}
}

// defaultValidator 使用 go-playground/validator 和 `validate` 标签校验普通
// 结构体；protobuf 请求则使用可选的 ProtoValidator。
type defaultValidator struct {
	structs        *playground.Validate
	protoValidator ProtoValidator
}

func newDefaultValidator() *defaultValidator {
	validator := playground.New()
	// playground 默认把 Go 字段名放进错误，例如 Username。客户端看到的是
	// json、query、path、header 或 form 标签里的名字，所以按这个顺序取第一个
	// 可用标签。标签值可以是 "display_name,omitempty"，逗号后是选项，只保留
	// 名字。"-" 表示该格式不暴露这个字段，继续看下一种标签。都没有时退回字段名。
	validator.RegisterTagNameFunc(func(field reflect.StructField) string {
		for _, source := range []string{"json", "query", "path", "header", "form"} {
			name := strings.SplitN(field.Tag.Get(source), ",", 2)[0]
			if name != "" && name != "-" {
				return name
			}
		}
		return field.Name
	})
	return &defaultValidator{structs: validator}
}

// Validate 在参数绑定之后、业务端点之前执行。先按 validate 标签检查每个字段；
// 标签没通过就直接返回，不再调用 ValidateRequest，避免在缺字段的对象上做
// 跨字段判断。标签通过后，请求结构如果实现了 RequestValidator，再执行它。
func (v *defaultValidator) Validate(ctx context.Context, request any) error {
	// 使用 protobuf 类型来实现的 HTTP/JSON 接口使用proto的校验器(即request和response是使用proto生成的)
	if message, ok := request.(proto.Message); ok {
		if v.protoValidator != nil {
			if err := v.protoValidator(ctx, message); err != nil {
				return err
			}
		}
	} else if err := v.structs.Struct(request); err != nil {
		return err
	}

	if validator, ok := request.(RequestValidator); ok {
		return validator.ValidateRequest(ctx)
	}
	return nil
}

// validationDetails 把 playground 的字段错误转换成响应里的 []Violation。
// 其他错误返回 nil。ValidateRequest 若要自己带详情，应返回 *ValidationError；
// 普通 error 不会在这里被拆成字段列表。
//
// 一条违规对应一个字段：Rule 是 validate 标签里的规则名，如 required、min；
// Parameter 是规则参数，如 min=8 里的 8。Namespace 形如
// "UpdateProfileRequest.profile.display_name"，第一段是 Go 结构体名，对客户端
// 没有意义，所以去掉。
func validationDetails(err error) any {
	var structViolations playground.ValidationErrors
	if errors.As(err, &structViolations) {
		details := make([]Violation, 0, len(structViolations))
		for _, violation := range structViolations {
			details = append(details, Violation{
				Field:     trimRootNamespace(violation.Namespace()),
				Rule:      violation.Tag(),
				Parameter: violation.Param(),
			})
		}
		return details
	}

	return nil
}

// normalizeValidationError 把校验失败统一成 *ValidationError。
// ValidateRequest 可以返回带 Details 的 *ValidationError，这里原样保留。
// playground 的错误则从中抽出字段列表，放进 Details。
func normalizeValidationError(err error) *ValidationError {
	var validation *ValidationError
	if errors.As(err, &validation) {
		return validation
	}
	return &ValidationError{
		Cause:   err,
		Details: validationDetails(err),
	}
}

// trimRootNamespace 去掉 playground 命名空间里的根结构体名。
// "UpdateProfileRequest.profile.display_name" 得到 "profile.display_name"。
// 没有 "." 时说明错误就落在根上，原样返回。
func trimRootNamespace(namespace string) string {
	if separator := strings.IndexByte(namespace, '.'); separator >= 0 {
		return namespace[separator+1:]
	}
	return namespace
}
