package http

import (
	"booking-system-api/internal/middleware"
	"booking-system-api/internal/service"
	"booking-system-api/internal/util"

	"github.com/gofiber/fiber/v2"
)

type AuthHandler struct {
	svc *service.AuthService
}

func NewAuthHandler(svc *service.AuthService) *AuthHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(r fiber.Router) {
	g := r.Group("/auth")
	g.Post("/register", h.Register_)
	g.Post("/login", h.Login)
	g.Post("/refresh", h.Refresh)
	g.Post("/logout", middleware.Auth(), h.Logout)
	g.Post("/forgot-password", h.ForgotPassword)
	g.Post("/verify-otp", h.VerifyOTP)
	g.Post("/reset-password", h.ResetPassword)
	g.Patch("/change-password", middleware.Auth(), h.ChangePassword)
	g.Get("/me", middleware.Auth(), h.Me)
}

func (h *AuthHandler) Register_(c *fiber.Ctx) error {
	var req service.RegisterRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Register(c.Context(), req)
	if err != nil {
		return err
	}
	return util.Created(c, "Registration successful", data)
}

func (h *AuthHandler) Login(c *fiber.Ctx) error {
	var req service.LoginRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.Login(c.Context(), req)
	if err != nil {
		return err
	}
	return util.OK(c, "Login successful", data)
}

func (h *AuthHandler) Refresh(c *fiber.Ctx) error {
	var req service.RefreshRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.RefreshToken(c.Context(), req.RefreshToken)
	if err != nil {
		return err
	}
	return util.OK(c, "Token refreshed", data)
}

func (h *AuthHandler) Logout(c *fiber.Ctx) error {
	var req service.LogoutRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	_ = h.svc.Logout(c.Context(), req.RefreshToken, middleware.GetUserID(c))
	return util.OK(c, "Logged out successfully", nil)
}

func (h *AuthHandler) ForgotPassword(c *fiber.Ctx) error {
	var req service.ForgotPasswordRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	_ = h.svc.ForgotPassword(c.Context(), req)
	return util.OK(c, "If the email exists, an OTP has been sent.", nil)
}

func (h *AuthHandler) VerifyOTP(c *fiber.Ctx) error {
	var req service.VerifyOTPRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	data, err := h.svc.VerifyOTP(c.Context(), req)
	if err != nil {
		return err
	}
	return util.OK(c, "OTP verified", data)
}

func (h *AuthHandler) ResetPassword(c *fiber.Ctx) error {
	var req service.ResetPasswordRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.svc.ResetPassword(c.Context(), req); err != nil {
		return err
	}
	return util.OK(c, "Password reset successfully", nil)
}

func (h *AuthHandler) ChangePassword(c *fiber.Ctx) error {
	var req service.ChangePasswordRequest
	if err := bindAndValidate(c, &req); err != nil {
		return err
	}
	if err := h.svc.ChangePassword(c.Context(), middleware.GetUserID(c), req); err != nil {
		return err
	}
	return util.OK(c, "Password changed", nil)
}

func (h *AuthHandler) Me(c *fiber.Ctx) error {
	data, err := h.svc.GetMe(c.Context(), middleware.GetUserID(c))
	if err != nil {
		return err
	}
	return util.OK(c, "Profile retrieved", data)
}
