package repository

import (
	"context"
	"database/sql"
	"strings"
)

// Nomor telepon user (kolom users."phoneNumber", migrasi 000021). Untuk supir
// & penjaga ruangan sumber utamanya tetap tabel masing-masing (menu Driver /
// Penjaga Ruangan bisa mengubahnya sendiri); kolom user jadi cadangan.
const userPhoneExpr = `CASE r.name
		WHEN 'DRIVER'      THEN COALESCE(NULLIF(d."phoneNumber", ''), u."phoneNumber", '')
		WHEN 'ROOM_KEEPER' THEN COALESCE(NULLIF(rk."phoneNumber", ''), u."phoneNumber", '')
		ELSE COALESCE(u."phoneNumber", '')
	END`

const userPhoneJoins = `
	JOIN roles r              ON r.id = u."roleId"
	LEFT JOIN drivers d       ON d."userId" = u.id
	LEFT JOIN room_keepers rk ON rk."userId" = u.id`

// SetUserPhone menyimpan nomor telepon user (kosong → NULL).
func (q *Queries) SetUserPhone(ctx context.Context, userID int32, phone string) error {
	phone = strings.TrimSpace(phone)
	_, err := q.db.ExecContext(ctx, `UPDATE users SET "phoneNumber" = $2 WHERE id = $1`,
		userID, sql.NullString{String: phone, Valid: phone != ""})
	return err
}

// GetUserPhone mengembalikan nomor telepon user ("" bila belum ada).
func (q *Queries) GetUserPhone(ctx context.Context, userID int32) (string, error) {
	var phone string
	err := q.db.QueryRowContext(ctx, `SELECT `+userPhoneExpr+` FROM users u`+userPhoneJoins+`
		WHERE u.id = $1`, userID).Scan(&phone)
	return phone, err
}

// GetUserPhones = GetUserPhone untuk banyak user sekaligus (daftar).
func (q *Queries) GetUserPhones(ctx context.Context, userIDs []int32) (map[int32]string, error) {
	out := make(map[int32]string, len(userIDs))
	if len(userIDs) == 0 {
		return out, nil
	}
	rows, err := q.db.QueryContext(ctx, `SELECT u.id, `+userPhoneExpr+` FROM users u`+userPhoneJoins+`
		WHERE u.id = ANY($1::int[])`, intArrayLiteral(userIDs))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id int32
		var phone string
		if err := rows.Scan(&id, &phone); err != nil {
			return nil, err
		}
		out[id] = phone
	}
	return out, rows.Err()
}
