package repository

import (
	"context"
	"database/sql"
)

// Hand-written (sqlc CLI unavailable in this environment) - mirrors what
// `sqlc generate` would produce from the matching entries in
// sql/query/room.sql.

func (q *Queries) SetRoomKeeper(ctx context.Context, roomID int32, roomKeeperID sql.NullInt32) (Room, error) {
	row := q.db.QueryRowContext(ctx,
		`UPDATE rooms SET "roomKeeperId" = $2 WHERE id = $1 RETURNING id, "resourceId", location, capacity, "photoUrl", "roomKeeperId"`,
		roomID, roomKeeperID,
	)
	var i Room
	err := row.Scan(&i.ID, &i.ResourceId, &i.Location, &i.Capacity, &i.PhotoUrl, &i.RoomKeeperId)
	return i, err
}

type GetRoomsByRoomKeeperIDRow struct {
	ID             int32          `json:"id"`
	ResourceId     int32          `json:"resourceId"`
	Location       string         `json:"location"`
	Capacity       int16          `json:"capacity"`
	PhotoUrl       sql.NullString `json:"photoUrl"`
	RoomKeeperId   sql.NullInt32  `json:"roomKeeperId"`
	ResourceName   string         `json:"resource_name"`
	ResourceStatus ResourceStatus `json:"resource_status"`
}

// GetRoomsByRoomKeeperID lists every room this room keeper is responsible
// for - unlike the vehicle<->driver fixed pairing this is N:1, so a room
// keeper can show up here for multiple rooms.
func (q *Queries) GetRoomsByRoomKeeperID(ctx context.Context, roomKeeperID int32) ([]GetRoomsByRoomKeeperIDRow, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT rm.id, rm."resourceId", rm.location, rm.capacity, rm."photoUrl", rm."roomKeeperId", r.name AS resource_name, r.status AS resource_status
		FROM rooms rm
		JOIN resources r ON r.id = rm."resourceId"
		WHERE rm."roomKeeperId" = $1
		ORDER BY r.name ASC`, roomKeeperID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []GetRoomsByRoomKeeperIDRow
	for rows.Next() {
		var i GetRoomsByRoomKeeperIDRow
		if err := rows.Scan(&i.ID, &i.ResourceId, &i.Location, &i.Capacity, &i.PhotoUrl, &i.RoomKeeperId, &i.ResourceName, &i.ResourceStatus); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
