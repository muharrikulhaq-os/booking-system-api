package service

import (
	"context"
	"database/sql"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type AuthService struct {
	q  repository.ExtendedQuerier
	db *sql.DB
}

func NewAuthService(db *sql.DB) *AuthService {
	return &AuthService{q: repository.New(db), db: db}
}

type LoginRequest struct {
	Email    string `json:"email"    validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type RegisterRequest struct {
	EmployeeID   string `json:"employeeId"   validate:"required"`
	Name         string `json:"name"         validate:"required"`
	Email        string `json:"email"        validate:"required,email"`
	Password     string `json:"password"     validate:"required,min=8"`
	RoleID       int32  `json:"roleId"       validate:"required"`
	DepartmentID int32  `json:"departmentId" validate:"required"`
}

type RefreshRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type LogoutRequest struct {
	RefreshToken string `json:"refreshToken" validate:"required"`
}

type ForgotPasswordRequest struct {
	Email string `json:"email" validate:"required,email"`
}

type VerifyOTPRequest struct {
	Email   string `json:"email"   validate:"required,email"`
	OTPCode string `json:"otpCode" validate:"required"`
}

type ResetPasswordRequest struct {
	ResetToken  string `json:"resetToken"  validate:"required"`
	NewPassword string `json:"newPassword" validate:"required,min=8"`
}

type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword" validate:"required"`
	NewPassword     string `json:"newPassword"     validate:"required,min=8"`
}

func (s *AuthService) Login(ctx context.Context, req LoginRequest) (map[string]any, error) {
	user, err := s.q.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, util.ErrWrongEmail
	}
	if !util.CheckPassword(req.Password, user.Password) {
		return nil, util.ErrWrongPassword
	}
	if !user.IsActive {
		return nil, util.ErrAccountInactive
	}

	accessToken, err := util.CreateAccessToken(int(user.ID), string(user.RoleName))
	if err != nil {
		return nil, err
	}
	refreshToken, expiresAt, err := util.CreateRefreshToken(int(user.ID))
	if err != nil {
		return nil, err
	}

	_, err = s.q.CreateRefreshToken(ctx, repository.CreateRefreshTokenParams{
		UserId:    user.ID,
		Token:     refreshToken,
		ExpiresAt: expiresAt,
	})
	if err != nil {
		return nil, err
	}

	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: user.ID, Valid: true},
		Action:      "LOGIN",
		EntityType:  "User",
		EntityId:    sql.NullInt32{Int32: user.ID, Valid: true},
		Description: sql.NullString{String: user.Name + " logged in", Valid: true},
	})

	return map[string]any{
		"accessToken":  accessToken,
		"refreshToken": refreshToken,
		"tokenType":    "Bearer",
		"user": map[string]any{
			"id":         user.ID,
			"employeeId": user.EmployeeId,
			"name":       user.Name,
			"email":      user.Email,
			"role":       string(user.RoleName),
			"department": user.DepartmentName,
		},
	}, nil
}

func (s *AuthService) Register(ctx context.Context, req RegisterRequest) (map[string]any, error) {
	if _, err := s.q.GetUserByEmail(ctx, req.Email); err == nil {
		return nil, util.ErrDuplicate
	}

	hashed, err := util.HashPassword(req.Password)
	if err != nil {
		return nil, err
	}

	user, err := s.q.CreateUser(ctx, repository.CreateUserParams{
		EmployeeId:   req.EmployeeID,
		Name:         req.Name,
		Email:        req.Email,
		Password:     hashed,
		IsActive:     true,
		RoleId:       req.RoleID,
		DepartmentId: req.DepartmentID,
	})
	if err != nil {
		return nil, err
	}

	full, _ := s.q.GetUserByID(ctx, user.ID)
	return map[string]any{
		"id":         full.ID,
		"employeeId": full.EmployeeId,
		"name":       full.Name,
		"email":      full.Email,
		"role":       string(full.RoleName),
		"department": full.DepartmentName,
	}, nil
}

func (s *AuthService) RefreshToken(ctx context.Context, refreshToken string) (map[string]any, error) {
	claims, err := util.ParseToken(refreshToken)
	if err != nil || claims.Type != "refresh" {
		return nil, util.ErrInvalidToken
	}

	stored, err := s.q.GetRefreshToken(ctx, repository.GetRefreshTokenParams{
		Token:  refreshToken,
		UserId: int32(claims.UserID),
	})
	if err != nil || stored.Revoked {
		return nil, util.ErrInvalidToken
	}

	if stored.ExpiresAt.Before(time.Now()) {
		_ = s.q.RevokeRefreshToken(ctx, repository.RevokeRefreshTokenParams{
			Token: refreshToken, UserId: int32(claims.UserID),
		})
		return nil, util.ErrInvalidToken
	}

	user, err := s.q.GetUserByID(ctx, int32(claims.UserID))
	if err != nil || !user.IsActive {
		return nil, util.ErrAccountInactive
	}

	accessToken, err := util.CreateAccessToken(int(user.ID), string(user.RoleName))
	if err != nil {
		return nil, err
	}

	return map[string]any{"accessToken": accessToken}, nil
}

func (s *AuthService) Logout(ctx context.Context, refreshToken string, userID int) error {
	_ = s.q.RevokeRefreshToken(ctx, repository.RevokeRefreshTokenParams{
		Token: refreshToken, UserId: int32(userID),
	})
	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: int32(userID), Valid: true},
		Action:      "LOGOUT",
		EntityType:  "User",
		EntityId:    sql.NullInt32{Int32: int32(userID), Valid: true},
		Description: sql.NullString{String: "User logged out", Valid: true},
	})
	return nil
}

func (s *AuthService) ForgotPassword(ctx context.Context, req ForgotPasswordRequest) error {
	user, err := s.q.GetUserByEmail(ctx, req.Email)
	if err != nil || !user.IsActive {
		return nil // silent
	}

	_ = s.q.InvalidatePreviousOTPs(ctx, user.ID)

	otp := util.GenerateOTP(6)
	expiry := util.OTPExpiry()
	_, _ = s.q.CreateOTP(ctx, repository.CreateOTPParams{
		UserId: user.ID, OtpCode: otp, ExpiresAt: expiry,
	})
	util.SendOTPEmail(user.Email, user.Name, otp)
	return nil
}

func (s *AuthService) VerifyOTP(ctx context.Context, req VerifyOTPRequest) (map[string]any, error) {
	user, err := s.q.GetUserByEmail(ctx, req.Email)
	if err != nil {
		return nil, util.ErrOTPInvalid
	}

	otp, err := s.q.GetValidOTP(ctx, repository.GetValidOTPParams{
		UserId: user.ID, OtpCode: req.OTPCode,
	})
	if err != nil {
		return nil, util.ErrOTPInvalid
	}
	if otp.ExpiresAt.Before(time.Now()) {
		return nil, util.ErrOTPExpired
	}

	_ = s.q.MarkOTPUsed(ctx, otp.ID)

	resetToken, err := util.CreateResetToken(int(user.ID))
	if err != nil {
		return nil, err
	}
	return map[string]any{"resetToken": resetToken}, nil
}

func (s *AuthService) ResetPassword(ctx context.Context, req ResetPasswordRequest) error {
	claims, err := util.ParseToken(req.ResetToken)
	if err != nil || claims.Type != "reset" {
		return util.ErrInvalidToken
	}

	hashed, err := util.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	return s.q.UpdateUserPassword(ctx, repository.UpdateUserPasswordParams{
		Password: hashed, ID: int32(claims.UserID),
	})
}

func (s *AuthService) ChangePassword(ctx context.Context, userID int, req ChangePasswordRequest) error {
	user, err := s.q.GetUserByID(ctx, int32(userID))
	if err != nil {
		return util.ErrNotFound
	}
	if !util.CheckPassword(req.CurrentPassword, user.Password) {
		return util.ErrUnauthorized
	}
	hashed, err := util.HashPassword(req.NewPassword)
	if err != nil {
		return err
	}
	return s.q.UpdateUserPassword(ctx, repository.UpdateUserPasswordParams{
		Password: hashed, ID: int32(userID),
	})
}

func (s *AuthService) GetMe(ctx context.Context, userID int) (map[string]any, error) {
	user, err := s.q.GetUserByID(ctx, int32(userID))
	if err != nil {
		return nil, util.ErrNotFound
	}
	return map[string]any{
		"id":           user.ID,
		"employeeId":   user.EmployeeId,
		"name":         user.Name,
		"email":        user.Email,
		"profilePhoto": nullStr(user.ProfilePhoto),
		"isActive":     user.IsActive,
		"role":         map[string]any{"id": user.RoleId, "name": string(user.RoleName)},
		"department":   map[string]any{"id": user.DepartmentId, "name": user.DepartmentName},
		"createdAt":    user.CreatedAt,
	}, nil
}

func nullStr(n sql.NullString) any {
	if n.Valid {
		return n.String
	}
	return nil
}
