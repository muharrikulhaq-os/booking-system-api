package service

import (
	"errors"

	"github.com/jackc/pgx/v5/pgconn"

	"booking-system-api/internal/util"
)

// inUseError mengubah pelanggaran foreign key (data masih dipakai riwayat
// lain, mis. booking) menjadi 409 dengan pesan jelas, bukan 500 (B18).
func inUseError(err error, msg string) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return util.NewError(409, msg, util.ErrConflict)
	}
	return err
}
