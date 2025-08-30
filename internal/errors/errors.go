package errors

import (
	"errors"
	"fmt"
)

var (
	ErrURLNotFound      = errors.New("URL not found")
	ErrURLAlreadyExists = errors.New("URL already exists")
	ErrInvalidURL       = errors.New("invalid URL")
	ErrInvalidShortCode = errors.New("invalid short code")
)

type ValidationError struct {
	Field   string
	Message string
}

func (e *ValidationError) Error() string {
	if e.Field != "" {
		return fmt.Sprintf("validation error in field '%s': %s", e.Field, e.Message)
	}
	return fmt.Sprintf("validation error: %s", e.Message)
}

func NewValidationError(field, message string) *ValidationError {
	return &ValidationError{
		Field:   field,
		Message: message,
	}
}

type BusinessError struct {
	Code    string
	Message string
	Cause   error
}

func (e *BusinessError) Error() string {
	if e.Cause != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Cause)
	}
	return e.Message
}

func (e *BusinessError) Unwrap() error {
	return e.Cause
}

func NewBusinessError(code, message string, cause error) *BusinessError {
	return &BusinessError{
		Code:    code,
		Message: message,
		Cause:   cause,
	}
}

var (
	ErrShortCodeGeneration = NewBusinessError("SHORT_CODE_GENERATION", "failed to generate unique short code", nil)
	ErrDatabaseOperation   = NewBusinessError("DATABASE_ERROR", "database operation failed", nil)
)

// IsValidationError проверяет является ли ошибка ошибкой валидации
func IsValidationError(err error) bool {
	var validationErr *ValidationError
	return errors.As(err, &validationErr)
}

// IsBusinessError проверяет является ли ошибка бизнес-ошибкой
func IsBusinessError(err error) bool {
	var businessErr *BusinessError
	return errors.As(err, &businessErr)
}

func GetValidationError(err error) *ValidationError {
	var validationErr *ValidationError
	if errors.As(err, &validationErr) {
		return validationErr
	}
	return nil
}

// GetBusinessError извлекает BusinessError из ошибки
func GetBusinessError(err error) *BusinessError {
	var businessErr *BusinessError
	if errors.As(err, &businessErr) {
		return businessErr
	}
	return nil
}
