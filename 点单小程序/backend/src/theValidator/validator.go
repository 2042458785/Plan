package theValidator

import (
	"fmt"
	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/validator/v10"
	"sync"
)

var once sync.Once
var registerError error

func RegisterCustValidator1() error {
	once.Do(func() {
		engine, ok := binding.Validator.Engine().(*validator.Validate)
		if !ok {
			registerError = fmt.Errorf("请求参数校验器类型不正确")
			return
		}
		registerError = engine.RegisterValidation("cents", func(field validator.FieldLevel) bool {
			value := field.Field().Int()
			return value >= 1 && value <= 1000000
		})
	})
	return registerError
}
