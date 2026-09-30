package service

import (
	"booking-system-api/internal/repository"
	"context"
	"database/sql"
)

type DashboardService struct {
	q          repository.ExtendedQuerier
	db         *sql.DB
	beforeRead func(ctx context.Context)
}

func NewDashboardService(db *sql.DB) *DashboardService {
	return &DashboardService{q: repository.New(db), db: db}
}

func (s *DashboardService) GetSummary(ctx context.Context) (any, error) {
	if s.beforeRead != nil {
		s.beforeRead(ctx)
	}
	return s.q.DashboardSummary(ctx)
}
