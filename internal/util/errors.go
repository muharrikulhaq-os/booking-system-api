package util

import "errors"

var (
	ErrNotFound          = errors.New("not found")
	ErrUnauthorized      = errors.New("unauthorized")
	ErrWrongEmail        = errors.New("email didnt match any accounts")
	ErrWrongPassword     = errors.New("incorrect password")
	ErrForbidden         = errors.New("forbidden")
	ErrConflict          = errors.New("conflict")
	ErrBadRequest        = errors.New("bad request")
	ErrDuplicate         = errors.New("already exists")
	ErrAccountInactive   = errors.New("account is inactive")
	ErrInvalidToken      = errors.New("invalid token")
	ErrOTPInvalid        = errors.New("invalid OTP code")
	ErrOTPExpired        = errors.New("OTP has expired")
	ErrBookingConflict   = errors.New("schedule conflict with an existing booking")
	ErrInvalidDateRange  = errors.New("end date must be after start date")
	ErrBookingNotPending = errors.New("booking is not in PENDING status")
	ErrSelfApproval      = errors.New("you cannot approve your own booking")
)

type AppError struct {
	Code    int
	Message string
	Err     error
}

func (e *AppError) Error() string { return e.Message }
func (e *AppError) Unwrap() error { return e.Err }

func NewError(code int, msg string, err error) *AppError {
	return &AppError{Code: code, Message: msg, Err: err}
}
