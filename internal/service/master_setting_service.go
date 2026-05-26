package service

import (
	"context"
	"database/sql"
	"fmt"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type MasterSettingService struct {
	q *repository.Queries
}

func NewMasterSettingService(db *sql.DB) *MasterSettingService {
	return &MasterSettingService{q: repository.New(db)}
}

type UpsertSettingRequest struct {
	Value       float64 `json:"value"       validate:"required"`
	Unit        string  `json:"unit"`
	Description string  `json:"description"`
}

func (s *MasterSettingService) List(ctx context.Context) (any, error) {
	return s.q.ListMasterSettings(ctx)
}

func (s *MasterSettingService) GetByKey(ctx context.Context, key string) (any, error) {
	ms, err := s.q.GetMasterSettingByKey(ctx, key)
	if err != nil {
		return nil, util.ErrNotFound
	}
	return ms, nil
}

func (s *MasterSettingService) Upsert(ctx context.Context, key string, req UpsertSettingRequest) (any, error) {
	return s.q.UpsertMasterSetting(ctx, repository.UpsertMasterSettingParams{
		Key:         key,
		Value:       fmt.Sprintf("%g", req.Value),
		Unit:        sql.NullString{String: req.Unit, Valid: req.Unit != ""},
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	})
}
