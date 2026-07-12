package service

import (
	"context"
	"database/sql"
	"strings"
	"strconv"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type BookingService struct {
	q     repository.ExtendedQuerier
	notif *NotificationService
}

func NewBookingService(db *sql.DB, notif *NotificationService) *BookingService {
	return &BookingService{q: repository.New(db), notif: notif}
}

type CreateBookingRequest struct {
	ResourceID       int32      `json:"resourceId" validate:"required"`
	StartDate        time.Time  `json:"startDate"  validate:"required"`
	EndDate          time.Time  `json:"endDate"    validate:"required"`
	Purpose          string     `json:"purpose"    validate:"required"`
	PassengerCount   int32      `json:"passengerCount" validate:"required,min=1"`
	DriverID         *int32     `json:"driverId"`
}

type ApproveBookingRequest struct {
	Note string `json:"note"`
}

type RejectBookingRequest struct {
	Note string `json:"note" validate:"required"`
}

type AssignVehicleRequest struct {
	DriverID  int32 `json:"driverId"  validate:"required"`
	VehicleID int32 `json:"vehicleId" validate:"required"`
}

type SubstituteResourceRequest struct {
	ResourceID int32  `json:"resourceId" validate:"required"`
	Note       string `json:"note"`
}

type RateDriverRequest struct {
	Rating int16  `json:"rating" validate:"required,min=1,max=5"`
	Review string `json:"review"`
}

type MergeBookingRequest struct {
	TargetBookingID int32      `json:"targetBookingId" validate:"required"`
	Reason          string     `json:"reason"`
	StartDate       *time.Time `json:"startDate"` // optional: override merged time window start
	EndDate         *time.Time `json:"endDate"`   // optional: override merged time window end
	DriverID        *int32     `json:"driverId"`  // optional: choose a driver for the merged booking
}

func serializeBookingRow(b repository.ListBookingsRow) map[string]any {
	out := map[string]any{
		"id":      b.ID,
		"status":  string(b.Status),
		"purpose": b.Purpose,
		"user": map[string]any{
			"id": b.UserId, "name": b.UserName,
			"employeeId": b.EmployeeId, "department": b.DepartmentName,
		},
		"resource": map[string]any{
			"id": b.ResourceId, "name": b.ResourceName,
			"type": string(b.ResourceType), "status": string(b.ResourceStatus),
		},
		"startDate":       b.StartDate,
		"endDate":         b.EndDate,
		"approvedBy":      nil,
		"approvedAt":      nullTime(b.ApprovedAt),
		"assignedAt":      nullTime(b.AssignedAt),
		"returnedAt":      nullTime(b.ReturnedAt),
		"createdAt":       b.CreatedAt,
		"updatedAt":       b.UpdatedAt,
		"assignedDriver":  nil,
		"assignedVehicle": nil,
		"isReassigned":    b.OriginalResourceId.Valid,
		"originalResource": nil,
		"hasMergeSuggestion": b.HasMergeSuggestion,
		"mergedIntoId":    nil,               // booking ini digabung KE booking mana (sekunder)
		"mergeCount":      b.MergeCount,       // jumlah booking yang digabung ke booking ini (primary)
		"isMerged":        b.MergedIntoId.Valid || b.MergeCount > 0,
	}
	if b.OriginalResourceId.Valid && b.OriginalResourceName.Valid {
		out["originalResource"] = map[string]any{
			"id": b.OriginalResourceId.Int32, "name": b.OriginalResourceName.String,
		}
	}
	if b.MergedIntoId.Valid {
		out["mergedIntoId"] = b.MergedIntoId.Int32
	}
	if b.ApproverName.Valid {
		out["approvedBy"] = map[string]any{"id": b.ApprovedById.Int32, "name": b.ApproverName.String}
	}
	if b.DriverID.Valid {
		out["assignedDriver"] = map[string]any{
			"id": b.DriverID.Int32, "name": b.DriverName.String, "phoneNumber": b.DriverPhone.String,
		}
	}
	if b.VehicleID.Valid {
		out["assignedVehicle"] = map[string]any{
			"id": b.VehicleID.Int32, "plateNumber": b.PlateNumber.String,
			"brand": b.Brand.String, "model": b.Model.String, "capacity": b.Capacity.Int16,
		}
	}
	return out
}

func serializeBookingByID(b repository.GetBookingByIDRow) map[string]any {
	out := map[string]any{
		"id":      b.ID,
		"status":  string(b.Status),
		"purpose": b.Purpose,
		"user": map[string]any{
			"id": b.UserId, "name": b.UserName,
			"employeeId": b.EmployeeId, "department": b.DepartmentName,
		},
		"resource": map[string]any{
			"id": b.ResourceId, "name": b.ResourceName,
			"type": string(b.ResourceType), "status": string(b.ResourceStatus),
		},
		"startDate":        b.StartDate,
		"endDate":          b.EndDate,
		"approvedBy":       nil,
		"approvedAt":       nullTime(b.ApprovedAt),
		"assignedAt":       nullTime(b.AssignedAt),
		"returnedAt":       nullTime(b.ReturnedAt),
		"createdAt":        b.CreatedAt,
		"updatedAt":        b.UpdatedAt,
		"assignedDriver":   nil,
		"assignedVehicle":  nil,
		"isReassigned":     b.OriginalResourceId.Valid,
		"hasMergeSuggestion": b.HasMergeSuggestion,
		"originalResource": nil,
	}
	if b.ApproverName.Valid {
		out["approvedBy"] = map[string]any{"id": b.ApprovedById.Int32, "name": b.ApproverName.String}
	}
	if b.DriverID.Valid {
		out["assignedDriver"] = map[string]any{
			"id": b.DriverID.Int32, "name": b.DriverName.String, "phoneNumber": b.DriverPhone.String,
		}
	}
	if b.VehicleID.Valid {
		out["assignedVehicle"] = map[string]any{
			"id": b.VehicleID.Int32, "plateNumber": b.PlateNumber.String,
			"brand": b.Brand.String, "model": b.Model.String, "capacity": b.Capacity.Int16,
		}
	}
	if b.OriginalResourceId.Valid {
		out["originalResource"] = map[string]any{
			"id": b.OriginalResourceId.Int32,
		}
	}
	return out
}

func nullTime(n sql.NullTime) any {
	if n.Valid {
		return n.Time
	}
	return nil
}

func nullInt32(n sql.NullInt32) any {
	if n.Valid {
		return n.Int32
	}
	return nil
}

func (s *BookingService) List(ctx context.Context,
	page, limit int,
	userID *int32,
	status *string,
	resourceID *int32,
	resourceType *string,
	driverID *int32,
	startFrom, endTo *time.Time,
	currentUserID int,
	currentRole string,
	search *string,
) ([]map[string]any, int64, error) {
	// Auto-transition stale bookings on every list call (lightweight)
	_, _ = s.q.MarkOverdueBookings(ctx)  // ONGOING + endDate passed → OVERDUE
	_, _ = s.q.MarkExpiredBookings(ctx)  // APPROVED + endDate passed, never started → EXPIRED
	_, _ = s.q.MarkIgnoredBookings(ctx)  // PENDING + endDate passed, admin didn't respond → IGNORED

	params := repository.ListBookingsParams{
		Limit:  int32(limit),
		Offset: int32((page - 1) * limit),
	}

	switch currentRole {
	case "DRIVER":
		d, err := s.q.GetDriverByUserID(ctx, int32(currentUserID))
		if err != nil {
			params.UserID = sql.NullInt32{Int32: -1, Valid: true}
		} else {
			params.DriverID = sql.NullInt32{Int32: d.ID, Valid: true}
		}
	case "EMPLOYEE":
		params.UserID = sql.NullInt32{Int32: int32(currentUserID), Valid: true}
	default:
		if userID != nil {
			params.UserID = sql.NullInt32{Int32: *userID, Valid: true}
		}
	}

	if status != nil {
		params.Status = repository.NullBookingStatus{
			BookingStatus: repository.BookingStatus(*status), Valid: true,
		}
	}
	if resourceID != nil {
		params.ResourceID = sql.NullInt32{Int32: *resourceID, Valid: true}
	}
	if resourceType != nil {
		params.ResourceType = repository.NullResourceType{
			ResourceType: repository.ResourceType(*resourceType), Valid: true,
		}
	}
	if driverID != nil {
		params.DriverID = sql.NullInt32{Int32: *driverID, Valid: true}
	}
	if startFrom != nil {
		params.StartFrom = sql.NullTime{Time: *startFrom, Valid: true}
	}
	if endTo != nil {
		params.EndTo = sql.NullTime{Time: *endTo, Valid: true}
	}
	if search != nil {
		params.Search = sql.NullString{String: *search, Valid: true}
	}

	rows, err := s.q.ListBookings(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountBookings(ctx, repository.CountBookingsParams{
		UserID: params.UserID, Status: params.Status,
		ResourceID: params.ResourceID, ResourceType: params.ResourceType,
		DriverID: params.DriverID, StartFrom: params.StartFrom, EndTo: params.EndTo,
		Search: params.Search,
	})

	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeBookingRow(r)
	}
	return out, total, nil
}

func (s *BookingService) GetByID(ctx context.Context, id int32, currentUserID int, currentRole string) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if currentRole == "EMPLOYEE" && int(b.UserId) != currentUserID {
		return nil, util.ErrForbidden
	}
	if currentRole == "DRIVER" {
		d, err := s.q.GetDriverByUserID(ctx, int32(currentUserID))
		if err != nil || !b.AssignedDriverId.Valid || b.AssignedDriverId.Int32 != d.ID {
			return nil, util.ErrForbidden
		}
	}
	return serializeBookingByID(b), nil
}

func (s *BookingService) Create(ctx context.Context, req CreateBookingRequest, userID int) (map[string]any, error) {
	if !req.EndDate.After(req.StartDate) {
		return nil, util.ErrInvalidDateRange
	}

	resource, err := s.q.GetResourceByID(ctx, req.ResourceID)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if resource.Status != repository.ResourceStatusAVAILABLE {
		return nil, util.NewError(409, "resource is not available", util.ErrConflict)
	}

	var driverID sql.NullInt32
	var vehicleID sql.NullInt32

	// Kendaraan yang dibooking (error → resource adalah ruangan, bukan kendaraan).
	bookedVehicleID, isVehicle := int32(0), false
	if vid, verr := s.q.GetVehicleIDByResourceID(ctx, req.ResourceID); verr == nil {
		bookedVehicleID, isVehicle = vid, true
	}

	if req.DriverID != nil {
		driverID = sql.NullInt32{Int32: *req.DriverID, Valid: true}
		// Supir yang sudah "memegang" kendaraan (aktif di booking lain) → pakai
		// kendaraannya (jalur merge). Supir kosong → nanti pakai kendaraan yang dibooking.
		if da, err := s.q.GetDriverCurrentAssignment(ctx, *req.DriverID); err == nil {
			vehicleID = sql.NullInt32{Int32: da.VehicleId, Valid: true}
		}
	} else if isVehicle {
		// Booking kendaraan tanpa memilih supir → tempel supir "kosong" yang senggang.
		// Bila tak ada → booking tetap PENDING tanpa supir (admin bisa menolak).
		if free, ferr := s.q.GetFreeDriver(ctx); ferr == nil {
			driverID = sql.NullInt32{Int32: free, Valid: true}
		}
	}

	// Booking kendaraan: bila kendaraan belum ditentukan (supir kosong / tanpa supir),
	// default ke kendaraan yang dibooking, supaya supir "memegangnya" saat disetujui.
	if !vehicleID.Valid && isVehicle {
		vehicleID = sql.NullInt32{Int32: bookedVehicleID, Valid: true}
	}

	// NOTE: PENDING booking conflict is no longer strictly blocking other PENDING bookings here,
	// but we might want to still check for APPROVED/ONGOING conflicts.
	// Actually, the new CheckBookingConflict already ignores PENDING if we want. Wait, CheckBookingConflict still checks PENDING. 
	// We'll leave it as is if they use the same resourceId, or maybe we remove the PENDING check in CheckBookingConflict? 
	// The plan says: "Jika status booking masih PENDING, jadwal supir dan kendaraan tersebut TETAP TERBUKA."
	// We should just not block it here. But CheckBookingConflict is resource-based (room/vehicle). We can just let it be for now since it's for resourceId (like room).

	b, err := s.q.CreateBooking(ctx, repository.CreateBookingParams{
		UserId:            int32(userID),
		ResourceId:        req.ResourceID,
		StartDate:         req.StartDate,
		EndDate:           req.EndDate,
		Purpose:           req.Purpose,
		PassengerCount:    req.PassengerCount,
		AssignedDriverId:  driverID,
		AssignedVehicleId: vehicleID,
	})
	if err != nil {
		return nil, err
	}

	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: int32(userID), Valid: true},
		Action:      "CREATE",
		EntityType:  "Booking",
		EntityId:    sql.NullInt32{Int32: b.ID, Valid: true},
		Description: sql.NullString{String: "Booking created", Valid: true},
	})

	full, _ := s.q.GetBookingByID(ctx, b.ID)
	return serializeBookingByID(full), nil
}

func (s *BookingService) Cancel(ctx context.Context, id int32, userID int, role string) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if role == "EMPLOYEE" && int(b.UserId) != userID {
		return nil, util.ErrForbidden
	}
	if b.Status != repository.BookingStatusPENDING {
		return nil, util.ErrBookingNotPending
	}
	if _, err = s.q.CancelBooking(ctx, id); err != nil {
		return nil, err
	}
	full, _ := s.q.GetBookingByID(ctx, id)
	return serializeBookingByID(full), nil
}

type ApproveBookingResponse struct {
	Data    map[string]any `json:"data"`
	Warning string         `json:"warning,omitempty"`
}

func (s *BookingService) Approve(ctx context.Context, id int32, req ApproveBookingRequest, approverID int) (ApproveBookingResponse, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return ApproveBookingResponse{}, util.ErrNotFound
	}
	if b.Status != repository.BookingStatusPENDING {
		return ApproveBookingResponse{}, util.ErrBookingNotPending
	}

	warningMsg := ""

	// Check if this booking has a vehicle assigned
	if b.AssignedVehicleId.Valid {
		v, err := s.q.GetVehicleByID(ctx, b.AssignedVehicleId.Int32)
		if err == nil {
			overlapCount, _ := s.q.GetOverlappingPassengerCount(ctx, repository.GetOverlappingPassengerCountParams{
				AssignedVehicleId: b.AssignedVehicleId,
				StartDate:         b.StartDate,
				EndDate:           b.EndDate,
				ID:                b.ID,
			})
			if int(overlapCount) + int(b.PassengerCount) > int(v.Capacity) {
				warningMsg = "Warning: Vehicle capacity overload! (Remaining capacity is negative). Please substitute vehicle if needed."
			}
		}
	}

	_, err = s.q.ApproveBooking(ctx, repository.ApproveBookingParams{
		ID:           id,
		ApprovedById: sql.NullInt32{Int32: int32(approverID), Valid: true},
	})
	if err != nil {
		return ApproveBookingResponse{}, err
	}

	// Kepemilikan kendaraan mengikuti siklus booking: begitu booking disetujui,
	// supir "memegang" kendaraan booking ini bila belum punya penugasan aktif.
	// (Supir yang sudah punya kendaraan = sedang aktif di booking lain / hasil merge,
	// jadi jangan ditimpa.) Penugasan ini dilepas lagi saat booking selesai.
	if b.AssignedDriverId.Valid && b.AssignedVehicleId.Valid {
		if _, aerr := s.q.GetDriverCurrentAssignment(ctx, b.AssignedDriverId.Int32); aerr != nil {
			_, _ = s.q.AssignDriverToVehicle(ctx, repository.AssignDriverToVehicleParams{
				DriverId:  b.AssignedDriverId.Int32,
				VehicleId: b.AssignedVehicleId.Int32,
			})
		}
	}

	_, _ = s.q.CreateApprovalLog(ctx, repository.CreateApprovalLogParams{
		BookingId:  id,
		ApproverId: int32(approverID),
		Action:     "APPROVE",
		Note:       sql.NullString{String: req.Note, Valid: req.Note != ""},
	})

	full, _ := s.q.GetBookingByID(ctx, id)
	
	// Notify user
	if s.notif != nil {
		s.notif.NotifyUser(full.UserId, "Your booking has been approved", map[string]any{
			"bookingId": full.ID,
		})
	}
	// Notify driver if assigned
	if full.DriverID.Valid && s.notif != nil {
		driver, err := s.q.GetDriverByID(ctx, full.DriverID.Int32)
		if err == nil {
			s.notif.NotifyDriver(driver.UserId, "You have been assigned to a new booking", map[string]any{
				"bookingId": full.ID,
			})
		}
	}

	return ApproveBookingResponse{
		Data:    serializeBookingByID(full),
		Warning: warningMsg,
	}, nil
}

func (s *BookingService) Reject(ctx context.Context, id int32, req RejectBookingRequest, approverID int) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if int(b.UserId) == approverID {
		return nil, util.ErrSelfApproval
	}
	if b.Status != repository.BookingStatusPENDING {
		return nil, util.ErrBookingNotPending
	}

	if _, err = s.q.RejectBooking(ctx, repository.RejectBookingParams{
		ID: id, ApprovedById: sql.NullInt32{Int32: int32(approverID), Valid: true},
	}); err != nil {
		return nil, err
	}

	_, _ = s.q.CreateApprovalLog(ctx, repository.CreateApprovalLogParams{
		BookingId:  id,
		ApproverId: int32(approverID),
		Action:     repository.ApprovalActionREJECTED,
		Note:       sql.NullString{String: req.Note, Valid: true},
	})
	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: int32(approverID), Valid: true},
		Action:      "REJECT",
		EntityType:  "Booking",
		EntityId:    sql.NullInt32{Int32: id, Valid: true},
		Description: sql.NullString{String: "Booking rejected: " + req.Note, Valid: true},
	})

	full, _ := s.q.GetBookingByID(ctx, id)
	go util.SendBookingStatusEmail(b.UserName, b.UserName, int(id), b.ResourceName, "REJECTED", req.Note)
	return serializeBookingByID(full), nil
}

func (s *BookingService) AssignVehicle(ctx context.Context, id int32, req AssignVehicleRequest, adminID int) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if b.Status != repository.BookingStatusAPPROVED {
		return nil, util.NewError(409, "only APPROVED bookings can have vehicle assigned", util.ErrForbidden)
	}
	if b.ResourceType != repository.ResourceTypeVEHICLE {
		return nil, util.NewError(400, "assignment only applies to vehicle bookings", util.ErrBadRequest)
	}

	driver, err := s.q.GetDriverByID(ctx, req.DriverID)
	if err != nil || !driver.IsActive {
		return nil, util.NewError(404, "active driver not found", util.ErrNotFound)
	}

	vehicle, err := s.q.GetVehicleByID(ctx, req.VehicleID)
	if err != nil {
		return nil, util.NewError(404, "vehicle not found", util.ErrNotFound)
	}

	count, _ := s.q.CheckVehicleConflict(ctx, repository.CheckVehicleConflictParams{
		AssignedVehicleId: sql.NullInt32{Int32: req.VehicleID, Valid: true},
		StartDate:         b.StartDate,
		EndDate:           b.EndDate,
		ID:                id,
	})
	if count > 0 {
		return nil, util.NewError(409, "vehicle is already assigned to another booking in this period", util.ErrConflict)
	}

	if err = s.q.AssignVehicleAndUpdateResource(ctx, id, req.DriverID, req.VehicleID, vehicle.ResourceId); err != nil {
		return nil, err
	}

	isReassigned := vehicle.ResourceId != b.ResourceId
	desc := "Driver and vehicle assigned to booking"
	if isReassigned {
		desc = "Vehicle reassigned: resource updated from original booking resource"
	}
	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: int32(adminID), Valid: true},
		Action:      "ASSIGN",
		EntityType:  "Booking",
		EntityId:    sql.NullInt32{Int32: id, Valid: true},
		Description: sql.NullString{String: desc, Valid: true},
	})

	full, _ := s.q.GetBookingByID(ctx, id)
	
	if s.notif != nil {
		s.notif.NotifyDriver(driver.UserId, "You have been assigned to a vehicle and booking", map[string]any{
			"bookingId": full.ID,
			"vehicleId": vehicle.ID,
		})
	}
	
	return serializeBookingByID(full), nil
}

func (s *BookingService) Start(ctx context.Context, id int32, userID int, role string) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if b.Status != repository.BookingStatusAPPROVED {
		return nil, util.NewError(409, "only APPROVED bookings can be started", util.ErrForbidden)
	}

	switch role {
	case "DRIVER":
		if b.ResourceType != repository.ResourceTypeVEHICLE {
			return nil, util.NewError(400, "driver can only start vehicle bookings", util.ErrBadRequest)
		}
		d, err := s.q.GetDriverByUserID(ctx, int32(userID))
		if err != nil || !b.AssignedDriverId.Valid || b.AssignedDriverId.Int32 != d.ID {
			return nil, util.ErrForbidden  
		}
	case "ROOM_KEEPER":
		if b.ResourceType != repository.ResourceTypeROOM {
			return nil, util.NewError(400, "room keeper can only start room bookings", util.ErrBadRequest)
		}
		rk, err := s.q.GetRoomKeeperByUserID(ctx, int32(userID))
		if err != nil || !rk.IsActive {
			return nil, util.ErrForbidden
		}
	case "ADMIN":
		// admin can start any type
	default:
		return nil, util.ErrForbidden
	}

	now := time.Now().UTC()
	if now.Before(b.StartDate) {
		return nil, util.NewError(400, "cannot start before scheduled time", util.ErrBadRequest)
	}
	if now.After(b.EndDate) {
		return nil, util.NewError(400, "booking period has already ended", util.ErrBadRequest)
	}

	if _, err = s.q.StartBooking(ctx, id); err != nil {
		return nil, err
	}
	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: int32(userID), Valid: true},
		Action:      "START",
		EntityType:  "Booking",
		EntityId:    sql.NullInt32{Int32: id, Valid: true},
		Description: sql.NullString{String: "Booking started", Valid: true},
	})

	full, _ := s.q.GetBookingByID(ctx, id)
	return serializeBookingByID(full), nil
}

func (s *BookingService) Complete(ctx context.Context, id int32, userID int, role string) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if b.Status != repository.BookingStatusONGOING && b.Status != repository.BookingStatusOVERDUE {
		return nil, util.NewError(409, "booking must be ONGOING or OVERDUE to complete", util.ErrForbidden)
	}
	if role == "ROOM_KEEPER" {
		if b.ResourceType != repository.ResourceTypeROOM {
			return nil, util.NewError(400, "room keeper can only complete room bookings", util.ErrBadRequest)
		}
		rk, err := s.q.GetRoomKeeperByUserID(ctx, int32(userID))
		if err != nil || !rk.IsActive {
			return nil, util.ErrForbidden
		}
	}

	if _, err = s.q.CompleteBooking(ctx, id); err != nil {
		return nil, err
	}

	// Lepas kepemilikan kendaraan supir bila ia tak punya booking aktif lain.
	// (Kalau masih ada booking APPROVED/ONGOING lain — mis. hasil merge di
	// kendaraan yang sama — kepemilikan dipertahankan.)
	if b.AssignedDriverId.Valid {
		other, _ := s.q.CountActiveBookingsByDriver(ctx, b.AssignedDriverId.Int32, b.ID)
		if other == 0 {
			_ = s.q.ReleaseDriver(ctx, b.AssignedDriverId.Int32)
		}
	}

	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: int32(userID), Valid: true},
		Action:      "COMPLETE",
		EntityType:  "Booking",
		EntityId:    sql.NullInt32{Int32: id, Valid: true},
		Description: sql.NullString{String: "Booking completed", Valid: true},
	})

	full, _ := s.q.GetBookingByID(ctx, id)
	return serializeBookingByID(full), nil
}

func (s *BookingService) RateDriver(ctx context.Context, bookingID int32, req RateDriverRequest, userID int) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if b.Status != repository.BookingStatusCOMPLETED {
		return nil, util.NewError(400, "can only rate completed bookings", util.ErrBadRequest)
	}
	if int(b.UserId) != userID {
		return nil, util.ErrForbidden
	}
	if b.ResourceType != repository.ResourceTypeVEHICLE {
		return nil, util.NewError(400, "driver rating is only for vehicle bookings", util.ErrBadRequest)
	}
	if !b.AssignedDriverId.Valid {
		return nil, util.NewError(400, "no driver was assigned to this booking", util.ErrBadRequest)
	}
	if _, err = s.q.GetDriverRatingByBooking(ctx, bookingID); err == nil {
		return nil, util.NewError(409, "you have already rated this booking", util.ErrConflict)
	}

	r, err := s.q.CreateDriverRating(ctx, repository.CreateDriverRatingParams{
		BookingId: bookingID,
		DriverId:  b.AssignedDriverId.Int32,
		RatedById: int32(userID),
		Rating:    req.Rating,
		Review:    sql.NullString{String: req.Review, Valid: req.Review != ""},
	})
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"id":        r.ID,
		"bookingId": r.BookingId,
		"driverId":  r.DriverId,
		"rating":    r.Rating,
		"review":    nullStr(r.Review),
		"createdAt": r.CreatedAt,
	}, nil
}

func (s *BookingService) GetDriverRatings(ctx context.Context, driverID int32) (map[string]any, error) {
	ratings, err := s.q.GetDriverRatings(ctx, driverID)
	if err != nil {
		return nil, err
	}
	var total float64
	items := make([]map[string]any, len(ratings))
	for i, r := range ratings {
		total += float64(r.Rating)
		items[i] = map[string]any{
			"id":        r.ID,
			"rating":    r.Rating,
			"review":    nullStr(r.Review),
			"ratedBy":   map[string]any{"id": r.RatedById, "name": r.RatedByName},
			"createdAt": r.CreatedAt,
		}
	}
	var avg any
	if len(ratings) > 0 {
		avg = total / float64(len(ratings))
	}
	return map[string]any{
		"driverId":      driverID,
		"totalRatings":  len(ratings),
		"averageRating": avg,
		"ratings":       items,
	}, nil
}

// GetBookingDriverRating returns the rating submitted for a single booking, or a 404
// error when the booking has not been rated. Kept lightweight so the FE can poll it
// to decide between "beri rating" prompt and showing the submitted rating.
func (s *BookingService) GetBookingDriverRating(ctx context.Context, bookingID int32) (map[string]any, error) {
	r, err := s.q.GetDriverRatingByBooking(ctx, bookingID)
	if err != nil {
		return nil, util.NewError(404, "booking has not been rated", util.ErrNotFound)
	}
	return map[string]any{
		"id":        r.ID,
		"bookingId": r.BookingId,
		"driverId":  r.DriverId,
		"rating":    r.Rating,
		"review":    nullStr(r.Review),
		"createdAt": r.CreatedAt,
	}, nil
}

func (s *BookingService) GetApprovalLog(ctx context.Context, bookingID int32) (any, error) {
	if _, err := s.q.GetBookingByID(ctx, bookingID); err != nil {
		return nil, util.ErrNotFound
	}
	logs, err := s.q.GetApprovalLogs(ctx, bookingID)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(logs))
	for i, l := range logs {
		out[i] = map[string]any{
			"id":        l.ID,
			"approver":  map[string]any{"id": l.ApproverId, "name": l.ApproverName},
			"action":    string(l.Action),
			"note":      nullStr(l.Note),
			"createdAt": l.CreatedAt,
		}
	}
	return out, nil
}

func (s *BookingService) MarkOverdue(ctx context.Context) (int, error) {
	r1, err := s.q.MarkOverdueBookings(ctx)
	if err != nil {
		return 0, err
	}
	r2, _ := s.q.MarkExpiredBookings(ctx)
	r3, _ := s.q.MarkIgnoredBookings(ctx)
	return len(r1) + len(r2) + len(r3), nil
}

func (s *BookingService) SubstituteResource(ctx context.Context, id int32, req SubstituteResourceRequest, adminID int) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if b.Status != repository.BookingStatusPENDING {
		return nil, util.NewError(409, "resource substitution only allowed for PENDING bookings", util.ErrConflict)
	}
	if req.ResourceID == b.ResourceId {
		return nil, util.NewError(400, "new resource is the same as the current resource", util.ErrBadRequest)
	}

	newResource, err := s.q.GetResourceByID(ctx, req.ResourceID)
	if err != nil {
		return nil, util.NewError(404, "resource not found", util.ErrNotFound)
	}
	if newResource.Type != b.ResourceType {
		return nil, util.NewError(400, "resource type mismatch: cannot substitute with a different resource type", util.ErrBadRequest)
	}
	if newResource.Status != repository.ResourceStatusAVAILABLE {
		return nil, util.NewError(409, "new resource is not available", util.ErrConflict)
	}

	count, _ := s.q.CheckBookingConflict(ctx, repository.CheckBookingConflictParams{
		ResourceId: req.ResourceID,
		StartDate:  b.StartDate,
		EndDate:    b.EndDate,
	})
	if count > 0 {
		return nil, util.NewError(409, "new resource has a schedule conflict in this period", util.ErrConflict)
	}

	if _, err = s.q.UpdateBookingResource(ctx, repository.UpdateBookingResourceParams{
		ID:         id,
		ResourceId: req.ResourceID,
	}); err != nil {
		return nil, err
	}

	note := req.Note
	if note == "" {
		note = "Resource substituted by admin"
	}
	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: int32(adminID), Valid: true},
		Action:      "SUBSTITUTE_RESOURCE",
		EntityType:  "Booking",
		EntityId:    sql.NullInt32{Int32: id, Valid: true},
		Description: sql.NullString{String: note, Valid: true},
	})

	full, _ := s.q.GetBookingByID(ctx, id)
	return serializeBookingByID(full), nil
}

func (s *BookingService) GetActivity(ctx context.Context, id int32, callerID int, callerRole string) (any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	// Akses disamakan dengan GetByID: admin/room-keeper bebas, employee hanya
	// pemilik, driver hanya yang ditugaskan. (Sebelumnya cuma admin/pemilik →
	// driver kena 403 dan timeline hilang di detail booking.)
	if callerRole == "EMPLOYEE" && int(b.UserId) != callerID {
		return nil, util.ErrForbidden
	}
	if callerRole == "DRIVER" {
		d, dErr := s.q.GetDriverByUserID(ctx, int32(callerID))
		if dErr != nil || !b.AssignedDriverId.Valid || b.AssignedDriverId.Int32 != d.ID {
			return nil, util.ErrForbidden
		}
	}
	rows, err := s.q.GetBookingActivity(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = map[string]any{
			"id":          r.ID,
			"action":      r.Action,
			"description": nullStr(r.Description),
			"actor":       nullStr(r.UserName),
			"createdAt":   r.CreatedAt,
		}
	}
	return out, nil
}

func (s *BookingService) MergeBookings(ctx context.Context, primaryID int32, req MergeBookingRequest, adminID int) (map[string]any, error) {
	if primaryID == req.TargetBookingID {
		return nil, util.NewError(400, "cannot merge a booking with itself", util.ErrBadRequest)
	}

	primary, err := s.q.GetBookingByID(ctx, primaryID)
	if err != nil {
		return nil, util.NewError(404, "primary booking not found", util.ErrNotFound)
	}
	target, err := s.q.GetBookingByID(ctx, req.TargetBookingID)
	if err != nil {
		return nil, util.NewError(404, "target booking not found", util.ErrNotFound)
	}

	allowedStatuses := map[repository.BookingStatus]bool{
		repository.BookingStatusAPPROVED: true,
		repository.BookingStatusPENDING:  true,
	}
	if !allowedStatuses[primary.Status] {
		return nil, util.NewError(409, "primary booking must be PENDING or APPROVED", util.ErrConflict)
	}
	if !allowedStatuses[target.Status] {
		return nil, util.NewError(409, "target booking must be PENDING or APPROVED", util.ErrConflict)
	}
	if primary.ResourceType != repository.ResourceTypeVEHICLE {
		return nil, util.NewError(400, "merge is only supported for vehicle bookings", util.ErrBadRequest)
	}
	if target.ResourceType != repository.ResourceTypeVEHICLE {
		return nil, util.NewError(400, "target booking must also be a vehicle booking", util.ErrBadRequest)
	}

	alreadyMerged, _ := s.q.CheckBookingAlreadyMerged(ctx, primaryID, req.TargetBookingID)
	if alreadyMerged {
		return nil, util.NewError(409, "these bookings are already merged", util.ErrConflict)
	}

	// Determine effective time window for the primary booking.
	// Default: union of both bookings (earliest start, latest end).
	// Admin may override via request body.
	effectiveStart := primary.StartDate
	if target.StartDate.Before(effectiveStart) {
		effectiveStart = target.StartDate
	}
	effectiveEnd := primary.EndDate
	if target.EndDate.After(effectiveEnd) {
		effectiveEnd = target.EndDate
	}
	if req.StartDate != nil {
		effectiveStart = *req.StartDate
	}
	if req.EndDate != nil {
		effectiveEnd = *req.EndDate
	}
	if !effectiveEnd.After(effectiveStart) {
		return nil, util.NewError(400, "effective end date must be after start date", util.ErrBadRequest)
	}

	// Check that the expanded time window doesn't conflict with another booking on the same resource.
	conflictCount, _ := s.q.CheckBookingConflict(ctx, repository.CheckBookingConflictParams{
		ResourceId: primary.ResourceId,
		StartDate:  effectiveStart,
		EndDate:    effectiveEnd,
		ExcludeID:  sql.NullInt32{Int32: primaryID, Valid: true},
	})
	// The TARGET booking is the merge partner, not a real conflict. When target and
	// primary share the same resource (merging into the same vehicle) the target is
	// counted by CheckBookingConflict (which only excludes the primary) — discount it.
	if conflictCount > 0 &&
		target.ResourceId == primary.ResourceId &&
		target.StartDate.Before(effectiveEnd) &&
		target.EndDate.After(effectiveStart) {
		conflictCount--
	}
	if conflictCount > 0 {
		return nil, util.NewError(409, "the merged time window conflicts with another booking on this resource", util.ErrConflict)
	}

	// Update primary booking's time window if it changed.
	if !effectiveStart.Equal(primary.StartDate) || !effectiveEnd.Equal(primary.EndDate) {
		if err = s.q.UpdateBookingDates(ctx, primaryID, effectiveStart, effectiveEnd); err != nil {
			return nil, err
		}
	}

	// Update primary booking's driver if requested
	if req.DriverID != nil {
		driverID := sql.NullInt32{Int32: *req.DriverID, Valid: true}
		var vehicleID sql.NullInt32
		da, err := s.q.GetDriverCurrentAssignment(ctx, *req.DriverID)
		if err == nil {
			vehicleID = sql.NullInt32{Int32: da.VehicleId, Valid: true}
		}
		_, err = s.q.AssignVehicleToBooking(ctx, repository.AssignVehicleToBookingParams{
			ID:                primaryID,
			AssignedDriverId:  driverID,
			AssignedVehicleId: vehicleID,
		})
		if err != nil {
			return nil, err
		}
		primary.AssignedDriverId = driverID
		primary.AssignedVehicleId = vehicleID
	}

	// Move the merged (target) booking onto the PRIMARY's vehicle/resource so both
	// bookings ride ONE vehicle (hemat kendaraan): resourceId follows the primary
	// (target's original vehicle is freed on the calendar) and driver/vehicle are
	// inherited from the primary when it has them. Done BEFORE creating the merge
	// record so a failure here doesn't leave an "already merged" state; the update
	// is idempotent (re-running keeps resourceId + originalResourceId stable).
	driverID := int32(0)
	vehicleID := int32(0)
	if primary.AssignedDriverId.Valid {
		driverID = primary.AssignedDriverId.Int32
	}
	if primary.AssignedVehicleId.Valid {
		vehicleID = primary.AssignedVehicleId.Int32
	}
	if err = s.q.InheritMergeResourceDriverVehicle(ctx,
		req.TargetBookingID,
		primary.ResourceId,
		driverID, vehicleID,
		primary.AssignedDriverId.Valid, primary.AssignedVehicleId.Valid,
	); err != nil {
		return nil, err
	}

	merge, err := s.q.CreateBookingMerge(ctx, primaryID, req.TargetBookingID, int32(adminID), req.Reason)
	if err != nil {
		return nil, err
	}

	desc := "Booking merged with #" + itoa(req.TargetBookingID)
	if req.Reason != "" {
		desc += ": " + req.Reason
	}
	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: int32(adminID), Valid: true},
		Action:      "MERGE",
		EntityType:  "Booking",
		EntityId:    sql.NullInt32{Int32: primaryID, Valid: true},
		Description: sql.NullString{String: desc, Valid: true},
	})
	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: int32(adminID), Valid: true},
		Action:      "MERGE",
		EntityType:  "Booking",
		EntityId:    sql.NullInt32{Int32: req.TargetBookingID, Valid: true},
		Description: sql.NullString{String: "Booking merged into primary #" + itoa(primaryID), Valid: true},
	})

	return map[string]any{
		"mergeId":            merge.ID,
		"primaryBookingId":   merge.PrimaryBookingId,
		"mergedBookingId":    merge.MergedBookingId,
		"mergedBy":           adminID,
		"reason":             nullStr(merge.Reason),
		"effectiveStartDate": effectiveStart,
		"effectiveEndDate":   effectiveEnd,
		"createdAt":          merge.CreatedAt,
	}, nil
}

func (s *BookingService) GetMergeInfo(ctx context.Context, id int32, callerID int, callerRole string) (any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if callerRole == "EMPLOYEE" && int(b.UserId) != callerID {
		return nil, util.ErrForbidden
	}
	if callerRole == "DRIVER" {
		d, dErr := s.q.GetDriverByUserID(ctx, int32(callerID))
		if dErr != nil || !b.AssignedDriverId.Valid || b.AssignedDriverId.Int32 != d.ID {
			return nil, util.ErrForbidden
		}
	}

	merges, err := s.q.GetBookingMerges(ctx, id)
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(merges))
	for i, m := range merges {
		out[i] = map[string]any{
			"mergeId":          m.ID,
			"primaryBookingId": m.PrimaryBookingID,
			"mergedBookingId":  m.MergedBookingID,
			"isPrimary":        m.IsPrimary,
			"mergedBy":         m.MergedByName,
			"reason":           nullStr(m.Reason),
			"createdAt":        m.CreatedAt,
			"linkedBooking": map[string]any{
				"bookingId":  m.OtherBookingID,
				"userId":     m.OtherUserID,
				"userName":   m.OtherUserName,
				"employeeId": m.OtherEmployeeID,
				"department": m.OtherDepartment,
				"purpose":    m.OtherPurpose,
			},
		}
	}
	return out, nil
}

func (s *BookingService) SubmitReturnReport(
	ctx context.Context,
	bookingID int32,
	note, location string,
	userID int,
) error {
	b, err := s.q.GetBookingByID(ctx, bookingID)
	if err != nil {
		return util.ErrNotFound
	}
	if b.ResourceType != repository.ResourceTypeVEHICLE {
		return util.NewError(400, "return report only applies to vehicle bookings", util.ErrBadRequest)
	}
	if b.Status != repository.BookingStatusONGOING && b.Status != repository.BookingStatusOVERDUE {
		return util.NewError(409, "booking must be ONGOING or OVERDUE to submit return report", util.ErrForbidden)
	}

	d, err := s.q.GetDriverByUserID(ctx, int32(userID))
	if err != nil || !b.AssignedDriverId.Valid || b.AssignedDriverId.Int32 != d.ID {
		return util.ErrForbidden
	}

	if _, err = s.q.GetReturnReport(ctx, bookingID); err == nil {
		return util.NewError(409, "return report already submitted for this booking", util.ErrConflict)
	}

	if _, err = s.q.CreateReturnReport(ctx, bookingID, int32(userID), note, location); err != nil {
		return err
	}

	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: int32(userID), Valid: true},
		Action:      "SUBMIT_RETURN_REPORT",
		EntityType:  "Booking",
		EntityId:    sql.NullInt32{Int32: bookingID, Valid: true},
		Description: sql.NullString{String: "Driver submitted return report", Valid: true},
	})
	return nil
}

func (s *BookingService) GetReturnReport(ctx context.Context, bookingID int32, callerID int, callerRole string) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, bookingID)
	if err != nil {
		return nil, util.ErrNotFound
	}

	if callerRole == "DRIVER" {
		d, dErr := s.q.GetDriverByUserID(ctx, int32(callerID))
		if dErr != nil || !b.AssignedDriverId.Valid || b.AssignedDriverId.Int32 != d.ID {
			return nil, util.ErrForbidden
		}
	} else if callerRole != "ADMIN" {
		return nil, util.ErrForbidden
	}

	report, err := s.q.GetReturnReport(ctx, bookingID)
	if err != nil {
		return nil, util.ErrNotFound
	}

	attachments, _ := s.q.ListAttachmentsByBooking(ctx, sql.NullInt32{Int32: bookingID, Valid: true})
	photos := make([]map[string]any, 0)
	for _, a := range attachments {
		if a.Description.Valid && a.Description.String == "return_photo" {
			fp := a.FilePath
			if !strings.HasPrefix(fp, "/uploads/") {
				fp = "/uploads/" + fp
			}
			photos = append(photos, map[string]any{
				"id":       a.ID,
				"filePath": fp,
				"fileName": a.FileName,
				"fileType": a.FileType,
			})
		}
	}

	return map[string]any{
		"id":            report.ID,
		"bookingId":     report.BookingId,
		"submittedBy":   map[string]any{"id": report.SubmittedById, "name": report.SubmitterName},
		"note":          report.Note,
		"location":      report.Location,
		"submittedAt":   report.SubmittedAt,
		"photos":        photos,
	}, nil
}

func itoa(n int32) string {
	return strconv.FormatInt(int64(n), 10)
}

