package service

import (
	"context"
	"database/sql"
	"fmt"
	"strconv"
	"strings"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type MasterSettingService struct {
	q repository.ExtendedQuerier
}

func NewMasterSettingService(db *sql.DB) *MasterSettingService {
	return &MasterSettingService{q: repository.New(db)}
}

type UpsertSettingRequest struct {
	Value       any    `json:"value"       validate:"required"` // changed to any to accept string or float
	Unit        string  `json:"unit"`
	Description string  `json:"description"`
}

func (s *MasterSettingService) List(ctx context.Context) (any, error) {
	return s.q.ListMasterSettings(ctx)
}

func (s *MasterSettingService) ListFuelPrices(ctx context.Context) (any, error) {
	rows, err := s.q.ListMasterSettings(ctx)
	if err != nil {
		return nil, err
	}
	
	type FuelPriceResponse struct {
		Grade        string  `json:"grade"`
		PricePerUnit float64 `json:"pricePerUnit"`
		Unit         string  `json:"unit"`
		UpdatedAt    string  `json:"updatedAt"`
	}

	var results []FuelPriceResponse
	for _, r := range rows {
		// Only include fuel_price_*
		if len(r.Key) > 11 && r.Key[:11] == "fuel_price_" {
			grade := r.Key[11:]
			price := util.ParseStringToFloat64(r.Value)
			results = append(results, FuelPriceResponse{
				Grade:        grade,
				PricePerUnit: price,
				Unit:         r.Unit.String,
				// Dulu Format("...Z") pada waktu ber-zona lokal: jam WIB
				// ditempeli huruf Z sehingga terbaca 7 jam lebih maju.
				UpdatedAt: r.UpdatedAt.UTC().Format(time.RFC3339),
			})
		}
	}
	return results, nil
}

func (s *MasterSettingService) GetByKey(ctx context.Context, key string) (any, error) {
	ms, err := s.q.GetMasterSettingByKey(ctx, key)
	if err != nil {
		return nil, util.ErrNotFound
	}
	return ms, nil
}

func (s *MasterSettingService) Upsert(ctx context.Context, key string, req UpsertSettingRequest) (any, error) {
	valStr := strings.TrimSpace(fmt.Sprintf("%v", req.Value))

	// Menit mulai lebih awal (SPD / Non-SPD / ruangan): bilangan bulat 0–1440.
	if strings.HasPrefix(key, "booking_start_early_minutes_") {
		v, err := strconv.Atoi(valStr)
		if err != nil || v < 0 || v > 24*60 {
			return nil, util.NewError(400, "menit mulai lebih awal harus bilangan bulat 0 - 1440", util.ErrBadRequest)
		}
		valStr = strconv.Itoa(v)
	}

	// Klien biasanya hanya mengirim nilai — satuan & keterangan lama jangan
	// sampai terhapus (dulu ikut menjadi NULL setiap kali disimpan).
	if req.Unit == "" || req.Description == "" {
		if cur, err := s.q.GetMasterSettingByKey(ctx, key); err == nil {
			if req.Unit == "" && cur.Unit.Valid {
				req.Unit = cur.Unit.String
			}
			if req.Description == "" && cur.Description.Valid {
				req.Description = cur.Description.String
			}
		}
	}

	return s.q.UpsertMasterSetting(ctx, repository.UpsertMasterSettingParams{
		Key:         key,
		Value:       valStr,
		Unit:        sql.NullString{String: req.Unit, Valid: req.Unit != ""},
		Description: sql.NullString{String: req.Description, Valid: req.Description != ""},
	})
}
