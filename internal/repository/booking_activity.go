package repository

import (
	"context"
	"database/sql"
	"time"
)

type BookingActivityRow struct {
	ID          int32          `json:"id"`
	Action      string         `json:"action"`
	Description sql.NullString `json:"description"`
	UserName    sql.NullString `json:"userName"`
	CreatedAt   time.Time      `json:"createdAt"`
}

const getBookingActivity = `
SELECT al.id, al.action, al.description, u.name AS user_name, al."createdAt"
FROM audit_logs al
LEFT JOIN users u ON u.id = al."userId"
WHERE al."entityType" = 'Booking' AND al."entityId" = $1
ORDER BY al."createdAt" ASC
`

func (q *Queries) GetBookingActivity(ctx context.Context, bookingID int32) ([]BookingActivityRow, error) {
	rows, err := q.db.QueryContext(ctx, getBookingActivity, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var items []BookingActivityRow
	for rows.Next() {
		var i BookingActivityRow
		if err := rows.Scan(&i.ID, &i.Action, &i.Description, &i.UserName, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
