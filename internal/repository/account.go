package repository

import (
	"context"
	"database/sql"
	"errors"
)

// GetAccountState membaca status aktif & nama role akun terkini. Akun yang
// sudah dihapus dianggap tidak aktif.
func (q *Queries) GetAccountState(ctx context.Context, userID int32) (active bool, role string, err error) {
	err = q.db.QueryRowContext(ctx, `
		SELECT u."isActive", r.name
		FROM users u JOIN roles r ON r.id = u."roleId"
		WHERE u.id = $1`, userID).Scan(&active, &role)
	if errors.Is(err, sql.ErrNoRows) {
		return false, "", nil
	}
	return active, role, err
}
