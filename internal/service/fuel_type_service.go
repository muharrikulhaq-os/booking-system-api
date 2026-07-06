package service

import (
	"context"
	"database/sql"
	"fmt"
	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type FuelTypeService struct {
	q repository.Querier
}

func NewFuelTypeService(db *sql.DB) *FuelTypeService {
	return &FuelTypeService{q: repository.New(db)}
}

type FuelTypeRequest struct {
	Name         string  `json:"name" validate:"required"`
	Type         string  `json:"type" validate:"required"`
	Unit         string  `json:"unit" validate:"required"`
	DefaultPrice float64 `json:"defaultPrice"`
	IsActive     bool    `json:"isActive"`
}

func serializeFuelType(ft repository.FuelType) map[string]any {
	var defPrice float64
	if ft.DefaultPrice.Valid {
		defPrice = util.ParseStringToFloat64(ft.DefaultPrice.String)
	}
	return map[string]any{
		"id":           ft.ID,
		"name":         ft.Name,
		"type":         ft.Type,
		"unit":         ft.Unit,
		"defaultPrice": defPrice,
		"isActive":     ft.IsActive,
		"createdAt":    ft.CreatedAt,
		"updatedAt":    ft.UpdatedAt,
	}
}

func (s *FuelTypeService) List(ctx context.Context) ([]map[string]any, error) {
	rows, err := s.q.ListFuelTypes(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeFuelType(r)
	}
	return out, nil
}

func (s *FuelTypeService) Create(ctx context.Context, req FuelTypeRequest) (map[string]any, error) {
	var defPrice sql.NullString
	if req.DefaultPrice > 0 {
		defPrice = sql.NullString{String: fmt.Sprintf("%.2f", req.DefaultPrice), Valid: true}
	}

	ft, err := s.q.CreateFuelType(ctx, repository.CreateFuelTypeParams{
		Name:         req.Name,
		Type:         repository.FuelCategory(req.Type),
		Unit:         repository.FuelUnit(req.Unit),
		DefaultPrice: defPrice,
		IsActive:     req.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return serializeFuelType(ft), nil
}

func (s *FuelTypeService) Update(ctx context.Context, id int32, req FuelTypeRequest) (map[string]any, error) {
	if _, err := s.q.GetFuelType(ctx, id); err != nil {
		return nil, util.ErrNotFound
	}

	var defPrice sql.NullString
	if req.DefaultPrice > 0 {
		defPrice = sql.NullString{String: fmt.Sprintf("%.2f", req.DefaultPrice), Valid: true}
	}

	ft, err := s.q.UpdateFuelType(ctx, repository.UpdateFuelTypeParams{
		ID:           id,
		Name:         req.Name,
		Type:         repository.FuelCategory(req.Type),
		Unit:         repository.FuelUnit(req.Unit),
		DefaultPrice: defPrice,
		IsActive:     req.IsActive,
	})
	if err != nil {
		return nil, err
	}
	return serializeFuelType(ft), nil
}

func (s *FuelTypeService) Delete(ctx context.Context, id int32) error {
	if _, err := s.q.GetFuelType(ctx, id); err != nil {
		return util.ErrNotFound
	}
	return s.q.DeleteFuelType(ctx, id)
}
