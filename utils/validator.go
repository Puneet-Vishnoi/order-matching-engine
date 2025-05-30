package utils

import (
	"github.com/go-playground/validator/v10"
	"sync"
)

var (
	validate     *validator.Validate
	onceValidate sync.Once
)

func GetValidator() *validator.Validate {
	onceValidate.Do(func() {
		validate = validator.New()
	})
	return validate
}
