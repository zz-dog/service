package validator

import (
	"fmt"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ErrorMsg 将 validator.ValidationErrors 转换为中文提示
func ErrorMsg(err error) string {
	if validationErrors, ok := err.(validator.ValidationErrors); ok {

		var msgs []string
		for _, e := range validationErrors {
			msgs = append(msgs, translate(e))
		}
		return strings.Join(msgs, "；")
	}
	return err.Error()
}

func translate(fe validator.FieldError) string {
	fieldName := fe.Field()
	if name, ok := fieldNameMap[fieldName]; ok {
		fieldName = name
	}

	switch fe.Tag() {
	case "required":
		return fmt.Sprintf("%s不能为空", fieldName)
	case "min":
		return fmt.Sprintf("%s长度不能小于%s", fieldName, fe.Param())
	case "max":
		return fmt.Sprintf("%s长度不能大于%s", fieldName, fe.Param())
	case "len":
		return fmt.Sprintf("%s长度必须为%s", fieldName, fe.Param())
	case "email":
		return fmt.Sprintf("%s格式不正确", fieldName)
	default:
		return fmt.Sprintf("%s 校验失败", fieldName)
	}
}

var fieldNameMap = map[string]string{
	"Username": "用户名",
	"Password": "密码",
	"Phone":    "手机号",
	"Email":    "邮箱",
	"Nickname": "昵称",
}
