package spec

import "errors"

var (
	ErrEmptySpecName      = errors.New("规格维度名不能为空")
	ErrEmptySpecValueName = errors.New("规格值名不能为空")
	ErrSpecNotFound       = errors.New("规格维度不存在")
	ErrSpecValueNotFound  = errors.New("规格值不存在")
	ErrDuplicateSpecValue = errors.New("规格值已存在")
	ErrSpecAlreadyExists  = errors.New("规格维度已存在")
)
