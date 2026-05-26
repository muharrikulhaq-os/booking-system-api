package util

import "github.com/gofiber/fiber/v2"

type SuccessResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

type PaginatedResponse struct {
	Success    bool       `json:"success"`
	Message    string     `json:"message"`
	Data       any        `json:"data"`
	Pagination Pagination `json:"pagination"`
}

type Pagination struct {
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	TotalPages int   `json:"totalPages"`
}

type ErrorDetail struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type ErrorResponse struct {
	Success bool        `json:"success"`
	Message string      `json:"message"`
	Error   ErrorDetail `json:"error"`
}

func OK(c *fiber.Ctx, msg string, data any) error {
	return c.Status(fiber.StatusOK).JSON(SuccessResponse{Success: true, Message: msg, Data: data})
}

func Created(c *fiber.Ctx, msg string, data any) error {
	return c.Status(fiber.StatusCreated).JSON(SuccessResponse{Success: true, Message: msg, Data: data})
}

func Paginated(c *fiber.Ctx, msg string, data any, total int64, page, limit int) error {
	totalPages := int((total + int64(limit) - 1) / int64(limit))
	return c.Status(fiber.StatusOK).JSON(PaginatedResponse{
		Success: true,
		Message: msg,
		Data:    data,
		Pagination: Pagination{
			Total:      total,
			Page:       page,
			Limit:      limit,
			TotalPages: totalPages,
		},
	})
}
