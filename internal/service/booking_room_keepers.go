package service

import (
	"context"
	"database/sql"
	"strings"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

// Penjaga ruangan pada booking (migrasi 000024): satu ruangan bisa dijaga
// beberapa room keeper. Semuanya menerima booking, notifikasi, dan status
// yang sama - seperti satu akun supir yang dipakai bersama - dan pemesan
// boleh menilai tiap penjaga (tidak wajib).

// requireRoomHasKeeper: ruangan tanpa penjaga aktif tidak bisa dibooking -
// admin wajib memilih penjaga ruangannya dulu.
func requireRoomHasKeeper(ctx context.Context, q repository.ExtendedQuerier, resourceID int32) error {
	ok, err := q.RoomHasActiveKeeper(ctx, resourceID)
	if err != nil {
		return err
	}
	if !ok {
		return util.NewError(409,
			"ruangan ini belum punya penjaga ruangan aktif - admin perlu memilih penjaga ruangannya dulu",
			util.ErrConflict)
	}
	return nil
}

// roomBookingLabel: "Ruang Rapat A · Sen, 6 Okt 08:00 WIB · Budi".
func roomBookingLabel(b repository.GetBookingByIDRow) string {
	return bookingLabel(b) + " · " + b.UserName
}

// notifyRoomKeepers mengirim notifikasi yang sama ke semua penjaga aktif
// ruangan [resourceID], kecuali [exceptUserID] (pelaku aksinya sendiri).
func (s *BookingService) notifyRoomKeepers(ctx context.Context, resourceID int32, notifType, title, message string, bookingID, exceptUserID int32) {
	if s.notif == nil {
		return
	}
	keepers, err := s.q.ListRoomKeepersByResource(ctx, resourceID)
	if err != nil {
		return
	}
	for _, k := range keepers {
		if !k.IsActive || k.UserID == exceptUserID {
			continue
		}
		s.notif.Notify(k.UserID, notifType, title, message, map[string]any{"bookingId": bookingID})
	}
}

// notifyRoomBookingKeepers: notifyRoomKeepers khusus booking ruangan.
func (s *BookingService) notifyRoomBookingKeepers(ctx context.Context, b repository.GetBookingByIDRow, notifType, title, suffix string, exceptUserID int32) {
	if b.ResourceType != repository.ResourceTypeROOM {
		return
	}
	s.notifyRoomKeepers(ctx, b.ResourceId, notifType, title, roomBookingLabel(b)+" — "+suffix, b.ID, exceptUserID)
}

func roomKeeperPayload(k repository.RoomKeeperRef) map[string]any {
	return map[string]any{
		"id":          k.RoomKeeperID,
		"userId":      k.UserID,
		"name":        k.Name,
		"phoneNumber": k.PhoneNumber,
		"isActive":    k.IsActive,
	}
}

// attachRoomKeepers menambahkan "roomKeepers" (penjaga ruangan booking ini)
// ke detail booking ruangan.
func (s *BookingService) attachRoomKeepers(ctx context.Context, out map[string]any, b repository.GetBookingByIDRow) {
	if b.ResourceType != repository.ResourceTypeROOM {
		return
	}
	keepers, _ := s.q.ListRoomKeepersByResource(ctx, b.ResourceId)
	list := make([]map[string]any, 0, len(keepers))
	for _, k := range keepers {
		list = append(list, roomKeeperPayload(k))
	}
	out["roomKeepers"] = list
}

// ── Rating per penjaga ──────────────────────────────────────────────────────

type RoomKeeperRatingInput struct {
	RoomKeeperID int32  `json:"roomKeeperId"`
	Rating       int16  `json:"rating"`
	Review       string `json:"review"`
}

// RateRoomRequest: [Ratings] = nilai per penjaga (penjaga yang tidak dinilai
// boleh dilewati). Bentuk lama {rating, review} dari aplikasi versi lama
// tetap diterima: nilainya dipakai untuk semua penjaga yang belum dinilai.
type RateRoomRequest struct {
	Rating  int16                   `json:"rating"`
	Review  string                  `json:"review"`
	Ratings []RoomKeeperRatingInput `json:"ratings"`
}

func validStars(r int16) bool { return r >= 1 && r <= 5 }

func (s *BookingService) RateRoom(ctx context.Context, bookingID int32, req RateRoomRequest, actor AuditActor) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if b.Status != repository.BookingStatusCOMPLETED {
		return nil, util.NewError(400, "can only rate completed bookings", util.ErrBadRequest)
	}
	if b.UserId != actor.UserID {
		return nil, util.ErrForbidden
	}
	if b.ResourceType != repository.ResourceTypeROOM {
		return nil, util.NewError(400, "room rating is only for room bookings", util.ErrBadRequest)
	}
	roomID, err := s.q.GetRoomIDByResourceID(ctx, b.ResourceId)
	if err != nil {
		return nil, util.NewError(404, "room not found", util.ErrNotFound)
	}
	keepers, err := s.q.ListRoomKeepersByResource(ctx, b.ResourceId)
	if err != nil {
		return nil, err
	}
	existing, err := s.q.ListRoomRatingsByBooking(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	rated := map[int32]bool{}
	for _, e := range existing {
		if e.RoomKeeperID.Valid {
			rated[e.RoomKeeperID.Int32] = true
		}
	}
	byID := map[int32]repository.RoomKeeperRef{}
	for _, k := range keepers {
		byID[k.RoomKeeperID] = k
	}

	legacy := len(req.Ratings) == 0
	inputs := req.Ratings
	if legacy {
		if !validStars(req.Rating) {
			return nil, util.NewError(400, "rating harus 1 sampai 5", util.ErrBadRequest)
		}
		// Ruangan lama tanpa penjaga: tetap bisa dinilai sekali (tanpa penjaga).
		if len(keepers) == 0 {
			if len(existing) > 0 {
				return nil, util.NewError(409, "you have already rated this booking", util.ErrConflict)
			}
			inputs = []RoomKeeperRatingInput{{Rating: req.Rating, Review: req.Review}}
		} else {
			for _, k := range keepers {
				if !rated[k.RoomKeeperID] {
					inputs = append(inputs, RoomKeeperRatingInput{RoomKeeperID: k.RoomKeeperID, Rating: req.Rating, Review: req.Review})
				}
			}
			if len(inputs) == 0 {
				return nil, util.NewError(409, "you have already rated this booking", util.ErrConflict)
			}
		}
	} else {
		seen := map[int32]bool{}
		for _, in := range inputs {
			k, ok := byID[in.RoomKeeperID]
			switch {
			case !validStars(in.Rating):
				return nil, util.NewError(400, "rating harus 1 sampai 5", util.ErrBadRequest)
			case !ok:
				return nil, util.NewError(400, "penjaga yang dinilai bukan penjaga ruangan ini", util.ErrBadRequest)
			case seen[in.RoomKeeperID]:
				return nil, util.NewError(400, "satu penjaga hanya bisa dinilai sekali", util.ErrBadRequest)
			case rated[in.RoomKeeperID]:
				return nil, util.NewError(409, k.Name+" sudah Anda nilai", util.ErrConflict)
			}
			seen[in.RoomKeeperID] = true
		}
	}

	params := make([]repository.CreateRoomRatingParams, len(inputs))
	for i, in := range inputs {
		review := strings.TrimSpace(in.Review)
		params[i] = repository.CreateRoomRatingParams{
			BookingId:    bookingID,
			RoomId:       roomID,
			RoomKeeperId: sql.NullInt32{Int32: in.RoomKeeperID, Valid: in.RoomKeeperID != 0},
			RatedById:    actor.UserID,
			Rating:       in.Rating,
			Review:       sql.NullString{String: review, Valid: review != ""},
		}
	}
	created := make([]repository.RoomRating, 0, len(params))
	insert := func(q repository.ExtendedQuerier) error {
		for _, p := range params {
			r, err := q.CreateRoomRating(ctx, p)
			if err != nil {
				if isUniqueViolation(err) {
					return util.NewError(409, "penjaga ini baru saja dinilai - muat ulang halaman", util.ErrConflict)
				}
				return err
			}
			created = append(created, r)
		}
		return nil
	}
	if s.db == nil { // unit test (mock querier)
		err = insert(s.q)
	} else {
		err = withTx(ctx, s.db, func(q *repository.Queries) error { return insert(q) })
	}
	if err != nil {
		return nil, err
	}

	parts := make([]string, 0, len(inputs))
	for _, in := range inputs {
		name := "ruangan"
		if k, ok := byID[in.RoomKeeperID]; ok {
			name = k.Name
		}
		parts = append(parts, name+" "+itoa(int32(in.Rating))+" bintang")
	}
	logAudit(ctx, s.q, actor, "RATE_ROOM", "Booking", bookingID, "Penjaga ruangan dinilai: "+strings.Join(parts, ", "))
	if s.notif != nil {
		for _, in := range inputs {
			if k, ok := byID[in.RoomKeeperID]; ok {
				s.notif.Notify(k.UserID, "ROOM_KEEPER_RATED", "Penilaian diterima",
					"Anda menerima rating "+itoa(int32(in.Rating))+" bintang · "+roomBookingLabel(b),
					map[string]any{"bookingId": bookingID})
			}
		}
	}

	// Aplikasi versi lama membaca satu objek rating.
	if legacy {
		r := created[0]
		return map[string]any{
			"id":           r.ID,
			"bookingId":    r.BookingId,
			"roomId":       r.RoomId,
			"roomKeeperId": nullInt32(r.RoomKeeperId),
			"rating":       r.Rating,
			"review":       nullStr(r.Review),
			"createdAt":    r.CreatedAt,
		}, nil
	}
	return s.GetBookingRoomRatings(ctx, bookingID)
}

// GetBookingRoomRatings: penjaga ruangan booking ini beserta rating
// masing-masing (null = belum dinilai). Penjaga yang sudah tidak menjaga
// ruangan tetapi pernah dinilai di booking ini tetap ditampilkan.
func (s *BookingService) GetBookingRoomRatings(ctx context.Context, bookingID int32) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if b.ResourceType != repository.ResourceTypeROOM {
		return nil, util.NewError(400, "room rating is only for room bookings", util.ErrBadRequest)
	}
	keepers, err := s.q.ListRoomKeepersByResource(ctx, b.ResourceId)
	if err != nil {
		return nil, err
	}
	ratings, err := s.q.ListRoomRatingsByBooking(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	byKeeper := map[int32]repository.BookingRoomRatingRow{}
	for _, r := range ratings {
		if r.RoomKeeperID.Valid {
			byKeeper[r.RoomKeeperID.Int32] = r
		}
	}
	entry := func(id any, name string, r *repository.BookingRoomRatingRow) map[string]any {
		e := map[string]any{"roomKeeperId": id, "name": name, "rating": nil, "review": nil, "ratedAt": nil}
		if r != nil {
			e["rating"] = r.Rating
			e["review"] = nullStr(r.Review)
			e["ratedAt"] = r.CreatedAt
		}
		return e
	}
	items := make([]map[string]any, 0, len(keepers)+len(ratings))
	listed := map[int32]bool{}
	ratedCount := 0
	for _, k := range keepers {
		listed[k.RoomKeeperID] = true
		var rp *repository.BookingRoomRatingRow
		if r, ok := byKeeper[k.RoomKeeperID]; ok {
			rp = &r
			ratedCount++
		}
		items = append(items, entry(k.RoomKeeperID, k.Name, rp))
	}
	for i := range ratings {
		r := ratings[i]
		switch {
		case r.RoomKeeperID.Valid && listed[r.RoomKeeperID.Int32]:
			continue
		case r.RoomKeeperID.Valid:
			items = append(items, entry(r.RoomKeeperID.Int32, r.KeeperName.String, &r))
		default: // rating lama tanpa penjaga
			items = append(items, entry(nil, "Ruangan", &r))
		}
		ratedCount++
	}
	return map[string]any{
		"bookingId":  bookingID,
		"keepers":    items,
		"ratedCount": ratedCount,
		"total":      len(items),
	}, nil
}
