package validator

import (
	"fmt"
	"strings"

	"github.com/JahidNishat/payment-gateway/services/payment/internal/errors"
	"github.com/go-playground/validator/v10"
)

var validate = validator.New()

func ValidateStruct(s interface{}) error {
	if err := validate.Struct(s); err != nil {
		validationErrors := make(map[string][]string)
		for _, err := range err.(validator.ValidationErrors) {
			fieldName := strings.ToLower(err.Field())
			msg := translateError(err)
			validationErrors[fieldName] = append(validationErrors[fieldName], msg)
		}

		return &errors.ValidationError{
			Errors: validationErrors,
		}
	}
	return nil
}

func translateError(fe validator.FieldError) string {
	switch fe.Tag() {
	case "required":
		return "is required"
	case "min":
		return fmt.Sprintf("must be at least %s characters", fe.Param())
	case "max":
		return fmt.Sprintf("must be at most %s characters", fe.Param())
	case "len":
		return fmt.Sprintf("must be exactly %s characters", fe.Param())
	case "gt":
		return fmt.Sprintf("must be greater than %s", fe.Param())
	case "gte":
		return fmt.Sprintf("must be greater than or equal to %s", fe.Param())
	case "email":
		return "must be a valid email"
	default:
		return fe.Tag()
	}
}
