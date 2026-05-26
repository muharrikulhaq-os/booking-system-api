package service

import (
	"context"
	"database/sql"
	"time"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type BookingService struct {
	q  *repository.Queries
	db *sql.DB
}

func NewBookingService(db *sql.DB) *BookingService {
	return &BookingService{q: repository.New(db), db: db}
}

type CreateBookingRequest struct {
	ResourceID int32     `json:"resourceId" validate:"required"`
	StartDate  time.Time `json:"startDate"  validate:"required"`
	EndDate    time.Time `json:"endDate"    validate:"required"`
	Purpose    string    `json:"purpose"    validate:"required"`
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

type RateDriverRequest struct {
	Rating int16  `json:"rating" validate:"required,min=1,max=5"`
	Review string `json:"review"`
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
) ([]map[string]any, int64, error) {
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

	rows, err := s.q.ListBookings(ctx, params)
	if err != nil {
		return nil, 0, err
	}
	total, _ := s.q.CountBookings(ctx, repository.CountBookingsParams{
		UserID: params.UserID, Status: params.Status,
		ResourceID: params.ResourceID, ResourceType: params.ResourceType,
		DriverID: params.DriverID, StartFrom: params.StartFrom, EndTo: params.EndTo,
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

	// check resource
	resource, err := s.q.GetResourceByID(ctx, req.ResourceID)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if resource.Status != repository.ResourceStatusAVAILABLE {
		return nil, util.NewError(409, "resource is not available", util.ErrConflict)
	}

	// check conflict
	count, _ := s.q.CheckBookingConflict(ctx, repository.CheckBookingConflictParams{
		ResourceId: req.ResourceID,
		StartDate:  req.StartDate,
		EndDate:    req.EndDate,
	})
	if count > 0 {
		return nil, util.ErrBookingConflict
	}

	b, err := s.q.CreateBooking(ctx, repository.CreateBookingParams{
		UserId: int32(userID), ResourceId: req.ResourceID,
		StartDate: req.StartDate, EndDate: req.EndDate, Purpose: req.Purpose,
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

func (s *BookingService) Approve(ctx context.Context, id int32, req ApproveBookingRequest, approverID int) (map[string]any, error) {
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

	count, _ := s.q.CheckBookingConflict(ctx, repository.CheckBookingConflictParams{
		ResourceId: b.ResourceId, StartDate: b.StartDate, EndDate: b.EndDate,
		ExcludeID: sql.NullInt32{Int32: id, Valid: true},
	})
	if count > 0 {
		return nil, util.ErrBookingConflict
	}

	if _, err = s.q.ApproveBooking(ctx, repository.ApproveBookingParams{
		ID: id, ApprovedById: sql.NullInt32{Int32: int32(approverID), Valid: true},
	}); err != nil {
		return nil, err
	}

	_, _ = s.q.CreateApprovalLog(ctx, repository.CreateApprovalLogParams{
		BookingId:  id,
		ApproverId: int32(approverID),
		Action:     repository.ApprovalActionAPPROVED,
		Note:       sql.NullString{String: req.Note, Valid: req.Note != ""},
	})
	_, _ = s.q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: int32(approverID), Valid: true},
		Action:      "APPROVE",
		EntityType:  "Booking",
		EntityId:    sql.NullInt32{Int32: id, Valid: true},
		Description: sql.NullString{String: "Booking approved", Valid: true},
	})

	full, _ := s.q.GetBookingByID(ctx, id)
	go util.SendBookingStatusEmail(b.UserName, b.UserName, int(id), b.ResourceName, "APPROVED", req.Note)
	return serializeBookingByID(full), nil
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

	// check driver
	driver, err := s.q.GetDriverByID(ctx, req.DriverID)
	if err != nil || !driver.IsActive {
		return nil, util.NewError(404, "active driver not found", util.ErrNotFound)
	}

	// check vehicle conflict
	count, _ := s.q.CheckVehicleConflict(ctx, repository.CheckVehicleConflictParams{
		AssignedVehicleId: sql.NullInt32{Int32: req.VehicleID, Valid: true},
		StartDate:         b.StartDate,
		EndDate:           b.EndDate,
		ID:                id,
	})
	if count > 0 {
		return nil, util.NewError(409, "vehicle is already assigned to another booking in this period", util.ErrConflict)
	}

	if _, err = s.q.AssignVehicleToBooking(ctx, repository.AssignVehicleToBookingParams{
		ID:                id,
		AssignedDriverId:  sql.NullInt32{Int32: req.DriverID, Valid: true},
		AssignedVehicleId: sql.NullInt32{Int32: req.VehicleID, Valid: true},
	}); err != nil {
		return nil, err
	}

	full, _ := s.q.GetBookingByID(ctx, id)
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
	if b.ResourceType != repository.ResourceTypeVEHICLE {
		return nil, util.NewError(400, "only vehicle bookings can be started", util.ErrBadRequest)
	}
	if role == "DRIVER" {
		d, err := s.q.GetDriverByUserID(ctx, int32(userID))
		if err != nil || !b.AssignedDriverId.Valid || b.AssignedDriverId.Int32 != d.ID {
			return nil, util.ErrForbidden
		}
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
	full, _ := s.q.GetBookingByID(ctx, id)
	return serializeBookingByID(full), nil
}

func (s *BookingService) Complete(ctx context.Context, id int32) (map[string]any, error) {
	b, err := s.q.GetBookingByID(ctx, id)
	if err != nil {
		return nil, util.ErrNotFound
	}
	if b.Status != repository.BookingStatusONGOING && b.Status != repository.BookingStatusOVERDUE {
		return nil, util.NewError(409, "booking must be ONGOING or OVERDUE to complete", util.ErrForbidden)
	}
	if _, err = s.q.CompleteBooking(ctx, id); err != nil {
		return nil, err
	}
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
	rows, err := s.q.MarkOverdueBookings(ctx)
	if err != nil {
		return 0, err
	}
	return len(rows), nil
}
