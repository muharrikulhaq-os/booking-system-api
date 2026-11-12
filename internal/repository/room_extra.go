package repository

import (
	"context"
	"database/sql"
	"time"
)

// Hand-written (sqlc CLI unavailable in this environment) - mirrors what
// `sqlc generate` would produce from the matching entries in
// sql/query/room.sql.
//
// Penjaga ruangan: N:N lewat tabel room_keeper_rooms (migrasi 000024).
// rooms."roomKeeperId" sudah tidak dipakai.

// RoomKeeperRef: satu penjaga sebuah ruangan.
type RoomKeeperRef struct {
	RoomID       int32  `json:"roomId"`
	RoomKeeperID int32  `json:"roomKeeperId"`
	UserID       int32  `json:"userId"`
	Name         string `json:"name"`
	PhoneNumber  string `json:"phoneNumber"`
	// Aktif = penjaga & akun penggunanya sama-sama aktif.
	IsActive bool `json:"isActive"`
}

const roomKeeperRefSelect = `
	SELECT rkr."roomId", rk.id, u.id, u.name, COALESCE(NULLIF(rk."phoneNumber", ''), u."phoneNumber", ''),
	       (rk."isActive" AND u."isActive")
	FROM room_keeper_rooms rkr
	JOIN room_keepers rk ON rk.id = rkr."roomKeeperId"
	JOIN users u ON u.id = rk."userId"`

func (q *Queries) scanRoomKeeperRefs(ctx context.Context, query string, args ...any) ([]RoomKeeperRef, error) {
	rows, err := q.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []RoomKeeperRef{}
	for rows.Next() {
		var i RoomKeeperRef
		if err := rows.Scan(&i.RoomID, &i.RoomKeeperID, &i.UserID, &i.Name, &i.PhoneNumber, &i.IsActive); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// ListRoomKeepersOfRoom: penjaga satu ruangan (aktif maupun tidak).
func (q *Queries) ListRoomKeepersOfRoom(ctx context.Context, roomID int32) ([]RoomKeeperRef, error) {
	return q.scanRoomKeeperRefs(ctx, roomKeeperRefSelect+`
	WHERE rkr."roomId" = $1
	ORDER BY u.name ASC`, roomID)
}

// ListRoomKeepersByResource: penjaga ruangan dari resourceId booking.
func (q *Queries) ListRoomKeepersByResource(ctx context.Context, resourceID int32) ([]RoomKeeperRef, error) {
	return q.scanRoomKeeperRefs(ctx, roomKeeperRefSelect+`
	JOIN rooms rm ON rm.id = rkr."roomId"
	WHERE rm."resourceId" = $1
	ORDER BY u.name ASC`, resourceID)
}

// ListAllRoomKeeperRefs: semua pasangan ruangan-penjaga (untuk daftar ruangan).
func (q *Queries) ListAllRoomKeeperRefs(ctx context.Context) ([]RoomKeeperRef, error) {
	return q.scanRoomKeeperRefs(ctx, roomKeeperRefSelect+`
	ORDER BY u.name ASC`)
}

// SetRoomKeepers mengganti seluruh penjaga ruangan dengan [keeperIDs]
// (kosong = tanpa penjaga) dalam satu pernyataan.
func (q *Queries) SetRoomKeepers(ctx context.Context, roomID int32, keeperIDs []int32) error {
	_, err := q.db.ExecContext(ctx, `
		WITH del AS (
			DELETE FROM room_keeper_rooms
			WHERE "roomId" = $1 AND NOT ("roomKeeperId" = ANY($2::int[]))
		)
		INSERT INTO room_keeper_rooms ("roomId", "roomKeeperId")
		SELECT $1, k FROM unnest($2::int[]) AS k
		ON CONFLICT DO NOTHING`, roomID, intArrayLiteral(keeperIDs))
	return err
}

// IsRoomKeeperOfResource: apakah penjaga ini menjaga ruangan resource tsb.
func (q *Queries) IsRoomKeeperOfResource(ctx context.Context, roomKeeperID, resourceID int32) (bool, error) {
	var ok bool
	err := q.db.QueryRowContext(ctx, `
		SELECT EXISTS (
			SELECT 1 FROM room_keeper_rooms rkr
			JOIN rooms rm ON rm.id = rkr."roomId"
			WHERE rkr."roomKeeperId" = $1 AND rm."resourceId" = $2
		)`, roomKeeperID, resourceID).Scan(&ok)
	return ok, err
}

// RoomHasActiveKeeper: ruangan resource tsb punya minimal satu penjaga aktif.
func (q *Queries) RoomHasActiveKeeper(ctx context.Context, resourceID int32) (bool, error) {
	var ok bool
	err := q.db.QueryRowContext(ctx, `
		SELECT EXISTS (`+roomHasActiveKeeperCond+`)`, resourceID).Scan(&ok)
	return ok, err
}

// roomHasActiveKeeperCond: subquery EXISTS dengan $1 = resourceId (dipakai
// juga sebagai potongan SQL di filter bookable ListRooms).
const roomHasActiveKeeperCond = `
	SELECT 1 FROM room_keeper_rooms rkr
	JOIN rooms rm ON rm.id = rkr."roomId"
	JOIN room_keepers rk ON rk.id = rkr."roomKeeperId"
	JOIN users u ON u.id = rk."userId"
	WHERE rm."resourceId" = $1 AND rk."isActive" AND u."isActive"`

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

// GetRoomsByRoomKeeperID lists every room this room keeper guards.
func (q *Queries) GetRoomsByRoomKeeperID(ctx context.Context, roomKeeperID int32) ([]GetRoomsByRoomKeeperIDRow, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT rm.id, rm."resourceId", rm.location, rm.capacity, rm."photoUrl", rkr."roomKeeperId", r.name AS resource_name, r.status AS resource_status
		FROM room_keeper_rooms rkr
		JOIN rooms rm ON rm.id = rkr."roomId"
		JOIN resources r ON r.id = rm."resourceId"
		WHERE rkr."roomKeeperId" = $1
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

type RoomKeeperRoomRow struct {
	RoomKeeperID int32  `json:"room_keeper_id"`
	RoomID       int32  `json:"room_id"`
	ResourceName string `json:"resource_name"`
	Location     string `json:"location"`
}

// ListRoomsWithKeeper returns every (roomKeeperId, room) pair - batched
// lookup for RoomKeeperService.List.
func (q *Queries) ListRoomsWithKeeper(ctx context.Context) ([]RoomKeeperRoomRow, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT rkr."roomKeeperId", rm.id, r.name, rm.location
		FROM room_keeper_rooms rkr
		JOIN rooms rm ON rm.id = rkr."roomId"
		JOIN resources r ON r.id = rm."resourceId"
		ORDER BY r.name ASC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var items []RoomKeeperRoomRow
	for rows.Next() {
		var i RoomKeeperRoomRow
		if err := rows.Scan(&i.RoomKeeperID, &i.RoomID, &i.ResourceName, &i.Location); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

// BookingRoomRatingRow: rating penjaga untuk satu booking.
type BookingRoomRatingRow struct {
	RoomKeeperID sql.NullInt32  `json:"roomKeeperId"`
	KeeperName   sql.NullString `json:"keeperName"`
	Rating       int16          `json:"rating"`
	Review       sql.NullString `json:"review"`
	CreatedAt    time.Time      `json:"createdAt"`
}

// ListRoomRatingsByBooking: semua rating penjaga pada satu booking.
func (q *Queries) ListRoomRatingsByBooking(ctx context.Context, bookingID int32) ([]BookingRoomRatingRow, error) {
	rows, err := q.db.QueryContext(ctx, `
		SELECT rr."roomKeeperId", u.name, rr.rating, rr.review, rr."createdAt"
		FROM room_ratings rr
		LEFT JOIN room_keepers rk ON rk.id = rr."roomKeeperId"
		LEFT JOIN users u ON u.id = rk."userId"
		WHERE rr."bookingId" = $1
		ORDER BY rr."createdAt" ASC`, bookingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []BookingRoomRatingRow{}
	for rows.Next() {
		var i BookingRoomRatingRow
		if err := rows.Scan(&i.RoomKeeperID, &i.KeeperName, &i.Rating, &i.Review, &i.CreatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}
