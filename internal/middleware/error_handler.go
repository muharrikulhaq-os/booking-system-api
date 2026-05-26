package middleware

import (
	"errors"
	"log"

	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

func ErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError
	msg := "internal server error"

	var fiberErr *fiber.Error
	var appErr *util.AppError

	switch {
	case errors.As(err, &fiberErr):
		code = fiberErr.Code
		msg = fiberErr.Message
	case errors.As(err, &appErr):
		code = appErr.Code
		msg = appErr.Message
	case errors.Is(err, util.ErrNotFound):
		code, msg = fiber.StatusNotFound, err.Error()
	case errors.Is(err, util.ErrUnauthorized):
		code, msg = fiber.StatusUnauthorized, err.Error()
	case errors.Is(err, util.ErrForbidden):
		code, msg = fiber.StatusForbidden, err.Error()
	case errors.Is(err, util.ErrConflict),
		errors.Is(err, util.ErrBookingConflict):
		code, msg = fiber.StatusConflict, err.Error()
	case errors.Is(err, util.ErrDuplicate):
		code, msg = fiber.StatusConflict, err.Error()
	case errors.Is(err, util.ErrBadRequest),
		errors.Is(err, util.ErrInvalidDateRange),
		errors.Is(err, util.ErrBookingNotPending):
		code, msg = fiber.StatusBadRequest, err.Error()
	case errors.Is(err, util.ErrAccountInactive):
		code, msg = fiber.StatusForbidden, err.Error()
	case errors.Is(err, util.ErrInvalidToken),
		errors.Is(err, util.ErrOTPInvalid),
		errors.Is(err, util.ErrOTPExpired):
		code, msg = fiber.StatusUnauthorized, err.Error()
	case errors.Is(err, util.ErrSelfApproval):
		code, msg = fiber.StatusForbidden, err.Error()
	default:
		log.Printf("[ERROR] %v", err)
	}

	return c.Status(code).JSON(util.ErrorResponse{
		Success: false,
		Message: msg,
		Error:   util.ErrorDetail{Message: msg},
	})
}
