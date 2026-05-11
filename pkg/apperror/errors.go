package apperror

import (
	"errors"
	"net/http"
)

var (
	ErrBadRequest       = New(http.StatusBadRequest, "bad request")
	ErrUnauthorized     = New(http.StatusUnauthorized, "unauthorized")
	ErrForbidden        = New(http.StatusForbidden, "forbidden")
	ErrNotFound         = New(http.StatusNotFound, "resource not found")
	ErrConflict         = New(http.StatusConflict, "resource conflict")
	ErrValidation       = New(http.StatusBadRequest, "validation failed")
	ErrInternal         = New(http.StatusInternalServerError, "internal server error")
	ErrInvalidID        = New(http.StatusBadRequest, "invalid id")
	ErrInvalidAuth      = New(http.StatusUnauthorized, "invalid email or password")
	ErrEmailNotVerified = New(http.StatusForbidden, "email is not verified")
	ErrCarUnavailable   = New(http.StatusConflict, "car is not available")
	ErrDoubleBooking    = New(http.StatusConflict, "car is already rented for selected dates")
	ErrInvalidPayment   = New(http.StatusBadRequest, "invalid payment state")
	ErrPaymentExists    = New(http.StatusConflict, "payment already exists for this rental")
	ErrInvalidDate      = New(http.StatusBadRequest, "invalid date")
	ErrInvalidDateSpan  = New(http.StatusBadRequest, "end_date must be after or equal start_date")
)

type Error struct {
	Status  int
	Message string
	Cause   error
}

func New(status int, message string) *Error {
	return &Error{Status: status, Message: message}
}

func Wrap(base *Error, cause error) *Error {
	return &Error{Status: base.Status, Message: base.Message, Cause: cause}
}

func (e *Error) Error() string {
	return e.Message
}

func (e *Error) Unwrap() error {
	return e.Cause
}

func StatusCode(err error) int {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Status
	}

	return http.StatusInternalServerError
}

func Message(err error) string {
	var appErr *Error
	if errors.As(err, &appErr) {
		return appErr.Message
	}

	return ErrInternal.Message
}
