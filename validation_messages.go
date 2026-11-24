package ohmyhttp

import (
	"fmt"
	"reflect"
	"slices"
	"strconv"
	"strings"

	playground "github.com/go-playground/validator/v10"
)

type validationMessageTarget struct {
	path    string
	detail  *Violation
	matched bool
}

type validationMessageTargets []validationMessageTarget

func setValidationMessages(request any, rootName string, violations playground.ValidationErrors, details []Violation) {
	targets := make(validationMessageTargets, len(violations))
	for i, violation := range violations {
		targets[i] = validationMessageTarget{path: violation.StructNamespace(), detail: &details[i]}
	}
	slices.SortFunc(targets, func(a, b validationMessageTarget) int {
		return strings.Compare(a.path, b.path)
	})
	targets.apply(reflect.ValueOf(request), rootName, "")
}

// 从第一条具有该前缀的错误开始查找，没有相关错误就跳过整个分支。
func (targets validationMessageTargets) fromPrefix(prefix string) validationMessageTargets {
	start, _ := slices.BinarySearchFunc(targets, prefix, func(target validationMessageTarget, path string) int {
		return strings.Compare(target.path, path)
	})
	if start == len(targets) || !strings.HasPrefix(targets[start].path, prefix) {
		return nil
	}
	return targets[start:]
}

// dive 元素继承集合字段的标签，进入子结构体时改用子字段自己的标签。
// map 的键可能包含点或方括号，因此按实际键组成路径。
func (targets validationMessageTargets) apply(value reflect.Value, path string, tag reflect.StructTag) {
	targets = targets.fromPrefix(path)
	for len(targets) > 0 && targets[0].path == path {
		target := &targets[0]
		message := tag.Get("msg_" + target.detail.Rule)
		if message == "" {
			message = tag.Get("msg")
		}
		// validator 的 map 键与值可能产生相同路径。只有文案一致时才公开它。
		if target.matched && message != target.detail.Message {
			message = ""
		}
		target.detail.Message = message
		target.matched = true
		targets = targets[1:]
	}
	if len(targets) == 0 {
		return
	}
	if path != "" {
		children := targets.fromPrefix(path + ".")
		if len(children) == 0 {
			children = targets.fromPrefix(path + "[")
		}
		if len(children) == 0 {
			return
		}
		targets = children
	}
	for value.Kind() == reflect.Pointer || value.Kind() == reflect.Interface {
		value = value.Elem()
	}
	switch value.Kind() {
	case reflect.Struct:
		prefix := path
		if prefix != "" {
			prefix += "."
		}
		for i := 0; i < value.NumField(); i++ {
			field := value.Type().Field(i)
			if (field.IsExported() || field.Anonymous) && field.Tag.Get("validate") != "-" {
				targets.apply(value.Field(i), prefix+field.Name, field.Tag)
			}
		}
	case reflect.Array, reflect.Slice:
		for i := 0; i < value.Len(); i++ {
			targets.apply(value.Index(i), path+"["+strconv.Itoa(i)+"]", tag)
		}
	case reflect.Map:
		validateKeys := slices.Contains(strings.Split(tag.Get("validate"), ","), "keys")
		for iter := value.MapRange(); iter.Next(); {
			elementPath := fmt.Sprintf("%s[%v]", path, iter.Key())
			if validateKeys {
				targets.apply(iter.Key(), elementPath, tag)
			}
			targets.apply(iter.Value(), elementPath, tag)
		}
	}
}
