package ohmyhttp

import (
	"encoding"
	"fmt"
	"mime/multipart"
	"net/http"
	"net/url"
	"reflect"
	"strconv"
	"strings"
)

// 本文件负责基于 struct tag 的 Query、Header、Path 和 Form 参数绑定。

var (
	fileHeaderType      = reflect.TypeOf(multipart.FileHeader{})
	fileHeaderPointer   = reflect.TypeOf((*multipart.FileHeader)(nil))
	fileHeaderSlice     = reflect.TypeOf([]*multipart.FileHeader(nil))
	textUnmarshalerType = reflect.TypeOf((*encoding.TextUnmarshaler)(nil)).Elem()
)

func bindStructValues(target reflect.Value, tag string, values url.Values) error {
	// URL、Path 和 Form 中的原始值都是 string。walkTaggedFields 根据 tag
	// 找到目标字段，setTextValues 再负责 string -> Go 类型的转换。
	return walkTaggedFields(target, tag, func(name string, field reflect.Value) error {
		items, ok := values[name]
		if !ok || len(items) == 0 {
			return nil
		}
		return setTextValues(field, items)
	})
}

func clearTaggedFields(target reflect.Value, tags ...string) error {
	for target.Kind() == reflect.Pointer {
		if target.IsNil() {
			return nil
		}
		target = target.Elem()
	}
	if target.Kind() != reflect.Struct {
		return fmt.Errorf("tag clearing requires a struct, got %s", target.Kind())
	}
	for i := 0; i < target.NumField(); i++ {
		fieldType := target.Type().Field(i)
		if fieldType.PkgPath != "" {
			continue
		}
		field := target.Field(i)
		tagged := false
		for _, tag := range tags {
			name := strings.SplitN(fieldType.Tag.Get(tag), ",", 2)[0]
			if name != "" && name != "-" {
				tagged = true
				break
			}
		}
		if tagged {
			if !field.CanSet() {
				return errorsCannotSet(field)
			}
			field.Set(reflect.Zero(field.Type()))
			continue
		}
		kind := field.Kind()
		if kind == reflect.Pointer {
			if field.IsNil() {
				continue
			}
			kind = field.Elem().Kind()
		}
		if fieldType.Anonymous || kind == reflect.Struct {
			if err := clearTaggedFields(field, tags...); err != nil {
				return err
			}
		}
	}
	return nil
}

func bindStructHeaders(target reflect.Value, headers http.Header) error {
	return walkTaggedFields(target, "header", func(name string, field reflect.Value) error {
		items := headers.Values(name)
		if len(items) == 0 {
			return nil
		}
		return setTextValues(field, items)
	})
}

func bindMultipartFiles(
	target reflect.Value,
	files map[string][]*multipart.FileHeader,
) error {
	return walkTaggedFields(target, "form", func(name string, field reflect.Value) error {
		items := files[name]
		if len(items) == 0 {
			return nil
		}
		switch field.Type() {
		case fileHeaderPointer:
			field.Set(reflect.ValueOf(items[0]))
			return nil
		case fileHeaderSlice:
			field.Set(reflect.ValueOf(items))
			return nil
		case fileHeaderType:
			field.Set(reflect.ValueOf(*items[0]))
			return nil
		default:
			return nil
		}
	})
}

func walkTaggedFields(
	target reflect.Value,
	tag string,
	visit func(string, reflect.Value) error,
) error {
	for target.Kind() == reflect.Pointer {
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		target = target.Elem()
	}
	if target.Kind() != reflect.Struct {
		return fmt.Errorf("tag binding requires a struct, got %s", target.Kind())
	}

	for i := 0; i < target.NumField(); i++ {
		fieldType := target.Type().Field(i)
		if fieldType.PkgPath != "" {
			continue
		}
		field := target.Field(i)
		name := strings.SplitN(fieldType.Tag.Get(tag), ",", 2)[0]
		if name == "-" {
			continue
		}
		if name != "" {
			if err := visit(name, field); err != nil {
				return fmt.Errorf("field %s: %w", fieldType.Name, err)
			}
			continue
		}

		kind := field.Kind()
		if kind == reflect.Pointer && field.Type() != fileHeaderPointer {
			kind = field.Type().Elem().Kind()
		}
		if fieldType.Anonymous || kind == reflect.Struct {
			if field.Type() == fileHeaderType || field.Type() == fileHeaderPointer {
				continue
			}
			if err := walkTaggedFields(field, tag, visit); err != nil {
				return err
			}
		}
	}
	return nil
}

func setTextValues(field reflect.Value, values []string) error {
	if !field.CanSet() {
		return errorsCannotSet(field)
	}
	if field.Kind() == reflect.Slice && field.Type().Elem().Kind() != reflect.Uint8 {
		// ?tag=go&tag=http 绑定到 []string{"go", "http"}；其他 slice 也会
		// 对每个字符串逐个转换元素类型。
		slice := reflect.MakeSlice(field.Type(), 0, len(values))
		for _, value := range values {
			item := reflect.New(field.Type().Elem()).Elem()
			if err := setSingleTextValue(item, value); err != nil {
				return err
			}
			slice = reflect.Append(slice, item)
		}
		field.Set(slice)
		return nil
	}
	// 非 slice 字段收到重复参数时使用最后一个值。
	return setSingleTextValue(field, values[len(values)-1])
}

func setSingleTextValue(field reflect.Value, value string) error {
	// 指针字段按需分配，例如 *int 收到 "10" 后创建 int 并写入 10。
	if field.Kind() == reflect.Pointer {
		if field.IsNil() {
			field.Set(reflect.New(field.Type().Elem()))
		}
		return setSingleTextValue(field.Elem(), value)
	}
	// 自定义类型优先通过 encoding.TextUnmarshaler 解释文本，例如 UUID、日期等。
	if field.CanAddr() && field.Addr().Type().Implements(textUnmarshalerType) {
		return field.Addr().Interface().(encoding.TextUnmarshaler).UnmarshalText([]byte(value))
	}

	// URL/Header/Path/Form 都是文本，因此在这里统一转成目标标量类型。
	switch field.Kind() {
	case reflect.String:
		field.SetString(value)
	case reflect.Bool:
		parsed, err := strconv.ParseBool(value)
		if err != nil {
			return err
		}
		field.SetBool(parsed)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		parsed, err := strconv.ParseInt(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetInt(parsed)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		parsed, err := strconv.ParseUint(value, 10, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetUint(parsed)
	case reflect.Float32, reflect.Float64:
		parsed, err := strconv.ParseFloat(value, field.Type().Bits())
		if err != nil {
			return err
		}
		field.SetFloat(parsed)
	case reflect.Slice:
		if field.Type().Elem().Kind() != reflect.Uint8 {
			return fmt.Errorf("unsupported slice type %s", field.Type())
		}
		field.SetBytes([]byte(value))
	default:
		return fmt.Errorf("unsupported field type %s", field.Type())
	}
	return nil
}

func errorsCannotSet(field reflect.Value) error {
	return fmt.Errorf("field of type %s cannot be set", field.Type())
}
