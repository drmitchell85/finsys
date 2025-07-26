package utils

import (
	"errors"
	"fmt"
	"strings"
)

type ErrorCode string

const (
	// system errors
	ErrInternal     ErrorCode = "INTERNAL_ERROR"
	ErrValidation   ErrorCode = "VALIDATION_ERROR"
	ErrNotFound     ErrorCode = "NOT_FOUND"
	ErrUnauthorized ErrorCode = "UNAUTHORIZED"
	ErrForbidden    ErrorCode = "FORBIDDEN"

	// business logic errors
	ErrInsufficientFunds ErrorCode = "INSUFFICIENT_FUNDS"
	ErrAccountNotFound   ErrorCode = "ACCOUNT_NOT_FOUND"
	ErrDuplicateRequest  ErrorCode = "DUPLICATE_REQUEST"
	ErrUniqueConstraint  ErrorCode = "UNIQUE_CONSTRAINT_VIOLATION"
	// add more as needed
)

// AppError standardizes error handling across the application
type AppError struct {
	Code    ErrorCode `json:"code"`
	Message string    `json:"message"`
	Err     error     `json:"-"` // original error (not serialized)
}

func (e *AppError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s", e.Message, e.Err.Error())
	}
	return e.Message
}

func (e *AppError) Unwrap() error {
	return e.Err
}

// New creates a new AppError
func NewAppError(code ErrorCode, message string, err error) *AppError {
	// Check if err is already an AppError with the same code
	var appErr *AppError
	if errors.As(err, &appErr) && appErr.Code == code {
		// If the error already has the same code, just return it
		// This prevents double-wrapping
		return appErr
	}

	return &AppError{
		Code:    code,
		Message: message,
		Err:     err,
	}
}

// Helper functions for common errors
func NewValidationError(message string, err error) *AppError {
	return NewAppError(ErrValidation, message, err)
}

func NewNotFoundError(message string, err error) *AppError {
	return NewAppError(ErrNotFound, message, err)
}

func NewInternalError(err error) *AppError {
	return NewAppError(ErrInternal, "Internal server error", err)
}

func NewForbiddenError(message string, err error) *AppError {
	return NewAppError(ErrForbidden, message, err)
}

func NewConstraintError(err error) *AppError {
	errMsg := err.Error()
	if strings.Contains(errMsg, "duplicate key") || strings.Contains(errMsg, "unique constraint") {
		return NewAppError(ErrUniqueConstraint, "Duplicate record", err)
	}

	return NewInternalError(err)
}

// WrapError wraps an error with an AppError only if it's not already an AppError
// This ensures we don't double-wrap errors
func WrapError(err error, code ErrorCode, message string) *AppError {
	var appErr *AppError
	if errors.As(err, &appErr) {
		// Already an AppError, just return it
		return appErr
	}
	return NewAppError(code, message, err)
}

// GetAppError extracts an AppError from an error chain
func GetAppError(err error) (*AppError, bool) {
	var appErr *AppError
	if errors.As(err, &appErr) {
		return appErr, true
	}
	return nil, false
}
