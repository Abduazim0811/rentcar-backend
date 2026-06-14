package validator

import (
	"errors"
	"fmt"
	"reflect"
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
		field := fieldName(fieldErr)
		switch fieldErr.Tag() {
		case "required":
			messages = append(messages, fmt.Sprintf("%s is required", field))
		case "email":
			messages = append(messages, fmt.Sprintf("%s must be a valid email", field))
		case "len":
			messages = append(messages, fmt.Sprintf("%s must be exactly %s characters", field, fieldErr.Param()))
		case "min":
			messages = append(messages, fmt.Sprintf("%s must be at least %s%s", field, fieldErr.Param(), unit(fieldErr)))
		case "max":
			messages = append(messages, fmt.Sprintf("%s must be at most %s%s", field, fieldErr.Param(), unit(fieldErr)))
		case "gt":
			messages = append(messages, fmt.Sprintf("%s must be greater than %s", field, fieldErr.Param()))
		case "gte":
			messages = append(messages, fmt.Sprintf("%s must be greater than or equal to %s", field, fieldErr.Param()))
		case "lte":
			messages = append(messages, fmt.Sprintf("%s must be less than or equal to %s", field, fieldErr.Param()))
		case "oneof":
			messages = append(messages, fmt.Sprintf("%s must be one of: %s", field, fieldErr.Param()))
		case "url":
			messages = append(messages, fmt.Sprintf("%s must be a valid URL", field))
		default:
			messages = append(messages, fmt.Sprintf("%s is invalid", field))
		}
	}

	return strings.Join(messages, "; ")
}

func fieldName(fieldErr govalidator.FieldError) string {
	switch fieldErr.Field() {
	case "PlateNumber":
		return "plate number"
	case "DailyRate":
		return "daily rate"
	case "RefreshToken":
		return "refresh token"
	case "CarID":
		return "car"
	case "RentalID":
		return "rental"
	case "StartDate":
		return "pickup date"
	case "EndDate":
		return "return date"
	default:
		return splitFieldName(fieldErr.Field())
	}
}

func splitFieldName(field string) string {
	if field == "" {
		return "field"
	}

	var b strings.Builder
	for i, r := range field {
		if i > 0 && r >= 'A' && r <= 'Z' {
			b.WriteByte(' ')
		}
		b.WriteRune(r)
	}

	return strings.ToLower(b.String())
}

func unit(fieldErr govalidator.FieldError) string {
	if fieldErr.Kind() == reflect.String {
		return " characters"
	}

	return ""
}
