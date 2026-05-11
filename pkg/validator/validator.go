package validator

import (
	"errors"
	"fmt"
	"strings"

	govalidator "github.com/go-playground/validator/v10"
)

func Message(err error) string {
	var validationErrors govalidator.ValidationErrors
	if !errors.As(err, &validationErrors) {
		return "invalid request body"
	}

	messages := make([]string, 0, len(validationErrors))
	for _, fieldErr := range validationErrors {
		field := strings.ToLower(fieldErr.Field())
		switch fieldErr.Tag() {
		case "required":
			messages = append(messages, fmt.Sprintf("%s is required", field))
		case "email":
			messages = append(messages, fmt.Sprintf("%s must be a valid email", field))
		case "min":
			messages = append(messages, fmt.Sprintf("%s must be at least %s characters", field, fieldErr.Param()))
		case "gt":
			messages = append(messages, fmt.Sprintf("%s must be greater than %s", field, fieldErr.Param()))
		case "gte":
			messages = append(messages, fmt.Sprintf("%s must be greater than or equal to %s", field, fieldErr.Param()))
		case "oneof":
			messages = append(messages, fmt.Sprintf("%s must be one of: %s", field, fieldErr.Param()))
		default:
			messages = append(messages, fmt.Sprintf("%s is invalid", field))
		}
	}

	return strings.Join(messages, "; ")
}
