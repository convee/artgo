package artgo

import (
	"fmt"
	"github.com/go-playground/validator/v10"
	"reflect"
	"strings"
)

var structValidator = &defaultValidator{}

func EnableValidate() {
	structValidator.validate = validator.New()
	structValidator.enabled = true
}

type sliceValidateError []error

func (err sliceValidateError) Error() string {
	var errMsg []string
	for i, e := range err {
		if e == nil {
			continue
		}
		errMsg = append(errMsg, fmt.Sprintf("[%d]: %s", i, e.Error()))
	}
	return strings.Join(errMsg, "\n")
}

type defaultValidator struct {
	enabled  bool
	validate *validator.Validate
}

// ValidateStruct receives any kind of type, but only performed struct or pointer to struct type.
func (v *defaultValidator) ValidateStruct(obj interface{}) error {
	if obj == nil {
		return nil
	}
	value := reflect.ValueOf(obj)
	switch value.Kind() {
	case reflect.Ptr:
		return v.ValidateStruct(value.Elem().Interface())
	case reflect.Struct:
		return v.validateStruct(obj)
	case reflect.Slice, reflect.Array:
		count := value.Len()
		validateRet := make(sliceValidateError, 0)
		for i := 0; i < count; i++ {
			if err := v.ValidateStruct(value.Index(i).Interface()); err != nil {
				validateRet = append(validateRet, err)
			}
		}
		if len(validateRet) == 0 {
			return nil
		}
		return validateRet
	default:
		return nil
	}
}

func (v *defaultValidator) validateStruct(obj interface{}) error {
	return v.validate.Struct(obj)
}
