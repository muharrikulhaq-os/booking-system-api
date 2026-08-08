package repository

import "context"

// ListActiveUserIDsByRole returns the ids of all ACTIVE users having the given
// role name (e.g. "ADMIN", "ROOM_KEEPER"). Used to fan-out notifications.
func (q *Queries) ListActiveUserIDsByRole(ctx context.Context, role string) ([]int32, error) {
	const query = `
		SELECT u.id
		FROM users u
		JOIN roles r ON r.id = u."roleId"
		WHERE r.name = $1 AND u."isActive" = true`
	rows, err := q.db.QueryContext(ctx, query, role)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int32
	for rows.Next() {
		var id int32
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

// CountUnreadNotifications returns the number of unread notifications for a user.
func (q *Queries) CountUnreadNotifications(ctx context.Context, userID int32) (int64, error) {
	const query = `SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false`
	var n int64
	err := q.db.QueryRowContext(ctx, query, userID).Scan(&n)
	return n, err
}

// ─── FCM device tokens ───────────────────────────────────────────────────────

// UpsertDeviceToken stores/refreshes an FCM token for a user. A token is
// unique per device; re-registering it under another user reassigns it.
func (q *Queries) UpsertDeviceToken(ctx context.Context, userID int32, token, platform string) error {
	const query = `
		INSERT INTO device_tokens (user_id, token, platform)
		VALUES ($1, $2, $3)
		ON CONFLICT (token)
		DO UPDATE SET user_id = EXCLUDED.user_id, platform = EXCLUDED.platform, updated_at = NOW()`
	_, err := q.db.ExecContext(ctx, query, userID, token, platform)
	return err
}

// DeleteDeviceToken removes a token (logout / FCM reports it unregistered).
func (q *Queries) DeleteDeviceToken(ctx context.Context, token string) error {
	_, err := q.db.ExecContext(ctx, `DELETE FROM device_tokens WHERE token = $1`, token)
	return err
}

// ListDeviceTokensByUser returns all FCM tokens registered for a user.
func (q *Queries) ListDeviceTokensByUser(ctx context.Context, userID int32) ([]string, error) {
	rows, err := q.db.QueryContext(ctx, `SELECT token FROM device_tokens WHERE user_id = $1`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var tokens []string
	for rows.Next() {
		var t string
		if err := rows.Scan(&t); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}
