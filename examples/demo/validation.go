package demo

import playground "github.com/go-playground/validator/v10"

// ValidateUsername 演示字段级自定义校验规则。用户名必须以字母开头，并且只能包含
// 字母、数字和下划线。
func ValidateUsername(field playground.FieldLevel) bool {
	value := field.Field().String()
	if len(value) < 3 || len(value) > 32 {
		return false
	}
	for index, item := range value {
		if index == 0 && !isASCIILetter(item) {
			return false
		}
		if !isASCIILetter(item) && !isASCIIDigit(item) && item != '_' {
			return false
		}
	}
	return true
}

func isASCIILetter(value rune) bool {
	return value >= 'a' && value <= 'z' || value >= 'A' && value <= 'Z'
}

func isASCIIDigit(value rune) bool {
	return value >= '0' && value <= '9'
}
