package repository

import (
	"context"
	"database/sql"
	"time"
)

type ExtendedQuerier interface {
	Querier
	ListActiveUserIDsByRole(ctx context.Context, role string) ([]int32, error)
	CountUnreadNotifications(ctx context.Context, userID int32) (int64, error)
	UpsertDeviceToken(ctx context.Context, userID int32, token, platform string) error
	DeleteDeviceToken(ctx context.Context, token string) error
	ListDeviceTokensByUser(ctx context.Context, userID int32) ([]string, error)
	AssignVehicleAndUpdateResource(ctx context.Context, bookingID, driverID, vehicleID, vehicleResourceID int32) error
	CreateBookingMerge(ctx context.Context, primaryBookingID, mergedBookingID, mergedByID int32, reason string) (BookingMerge, error)
	GetBookingMerges(ctx context.Context, bookingID int32) ([]BookingMergeInfoRow, error)
	InheritMergeDriverVehicle(ctx context.Context, mergedBookingID, driverID, vehicleID int32, driverValid, vehicleValid bool) error
	InheritMergeResourceDriverVehicle(ctx context.Context, mergedBookingID, primaryResourceID, driverID, vehicleID int32, driverValid, vehicleValid bool) error
	UpdateBookingDates(ctx context.Context, bookingID int32, startDate, endDate time.Time) error
	CheckBookingAlreadyMerged(ctx context.Context, bookingA, bookingB int32) (bool, error)
	CountActiveBookingsByDriver(ctx context.Context, driverID, excludeBookingID int32) (int64, error)
	GetVehicleIDByResourceID(ctx context.Context, resourceID int32) (int32, error)
	GetFreeDriver(ctx context.Context) (int32, error)
	GetDriverActiveVehicleID(ctx context.Context, driverID int32) (int32, error)
	GetDriverHoldVehicleID(ctx context.Context, driverID int32) (int32, error)
	ListDriverConflictBookingIDs(ctx context.Context, driverID int32, start, end time.Time, excludeID int32) ([]int32, error)
	SubstituteBookingResource(ctx context.Context, bookingID, resourceID int32, vehicleID sql.NullInt32) error
	GetResourceStatusFacts(ctx context.Context, resourceID int32) (ResourceStatusFacts, error)
	PromoteDueMaintenance(ctx context.Context) (int64, error)
	ListDriverActiveBookingIDs(ctx context.Context, driverID int32) ([]int32, error)
	CountActiveResourceOverlap(ctx context.Context, resourceID int32, start, end time.Time, excludeID int32) (int64, error)
	ListVendors(ctx context.Context, p ListVendorsParams) ([]Vendor, error)
	GetVendor(ctx context.Context, id int32) (Vendor, error)
	CreateVendor(ctx context.Context, p VendorParams) (Vendor, error)
	UpdateVendor(ctx context.Context, id int32, p VendorParams) (Vendor, error)
	SetVendorActive(ctx context.Context, id int32, active bool) error
	DeleteVendor(ctx context.Context, id int32) error
	ListVehicleOwnership(ctx context.Context, vehicleIDs []int32) (map[int32]VehicleOwnership, error)
	GetVehicleOwnership(ctx context.Context, vehicleID int32) (VehicleOwnership, error)
	SetVehicleOwnership(ctx context.Context, vehicleID int32, ownership string, vendorID sql.NullInt32, contractNo sql.NullString) error
	ListMaintenanceRecords(ctx context.Context, p ListMaintenanceRecordsParams) ([]MaintenanceItem, int64, error)
	GetMaintenanceRecord(ctx context.Context, id int32) (MaintenanceItem, error)
	GetOpenMaintenanceID(ctx context.Context, vehicleID, excludeID int32) (int32, error)
	InsertMaintenance(ctx context.Context, p MaintenancePlan, sourceIssue sql.NullInt32, recordedBy int32) (int32, error)
	UpdateMaintenancePlan(ctx context.Context, id int32, p MaintenancePlan, allowed []string) (bool, error)
	TransitionMaintenance(ctx context.Context, id int32, from []string, to string, set string, args ...any) (bool, error)
	UpdateMaintenanceCost(ctx context.Context, id int32, estimated, actual, bearer sql.NullString) error
	DeleteMaintenanceRecord(ctx context.Context, id int32, allowed []string) (bool, error)
	NextDocumentNumber(ctx context.Context, key string) (int, error)
	GetDocumentSettings(ctx context.Context) (DocumentSettings, error)
	UpdateDocumentSettings(ctx context.Context, s DocumentSettings) error
	SetDocumentLogo(ctx context.Context, url sql.NullString) error
	ListMaintenanceDocuments(ctx context.Context, maintenanceID int32) ([]MaintenanceDocument, error)
	InsertMaintenanceDocument(ctx context.Context, d MaintenanceDocument) (int32, error)
	GetMaintenanceDocument(ctx context.Context, id int32) (MaintenanceDocument, error)
	DeleteMaintenanceDocument(ctx context.Context, id int32) error
	ListVehicleIssues(ctx context.Context, p ListVehicleIssuesParams) ([]VehicleIssue, int64, error)
	GetVehicleIssue(ctx context.Context, id int32) (VehicleIssue, error)
	InsertVehicleIssue(ctx context.Context, vehicleID int32, bookingID sql.NullInt32, reportedBy int32, description string, location sql.NullString, photos []string, canContinue bool) (int32, error)
	ResolveVehicleIssue(ctx context.Context, id int32, status string, handledBy int32, note sql.NullString, maintenanceID sql.NullInt32) (bool, error)
	CountOpenVehicleIssues(ctx context.Context) (int64, error)
	ListDriverVehicles(ctx context.Context, driverID int32) ([]DriverVehicle, error)
	CreateReturnReport(ctx context.Context, bookingID, submittedByID int32, note, location string, odometer sql.NullInt32) (BookingReturnReport, error)
	GetReturnReport(ctx context.Context, bookingID int32) (BookingReturnReportRow, error)
	SetBookingStartTrip(ctx context.Context, bookingID int32, odometer sql.NullInt32, location, photoURL sql.NullString) error
	GetRoomKeeperByUserID(ctx context.Context, userID int32) (RoomKeeper, error)
	GetBookingActivity(ctx context.Context, bookingID int32) ([]BookingActivityRow, error)
	ReportOverview(ctx context.Context, start, end time.Time) (OverviewRow, error)
	ReportBookingTrend(ctx context.Context, groupBy string, start, end time.Time) ([]BookingTrendRow, error)
	ReportBookingsByDepartment(ctx context.Context, start, end sql.NullTime) ([]BookingByDepartmentRow, error)
	ReportBookingsByResource(ctx context.Context, start, end sql.NullTime) ([]BookingByResourceRow, error)
	ReportApprovalPerformance(ctx context.Context, start, end sql.NullTime) (ApprovalPerformanceRow, error)
	ReportCostSummary(ctx context.Context, start, end sql.NullTime) (CostSummaryRow, error)
	ReportCostByVehicle(ctx context.Context, start, end sql.NullTime) ([]CostByVehicleRow, error)
	ReportCostByDepartment(ctx context.Context, start, end sql.NullTime) ([]CostByDepartmentRow, error)
	ReportCostTrend(ctx context.Context, groupBy string, start, end time.Time) ([]CostTrendRow, error)
	ReportDriverPerformance(ctx context.Context, start, end sql.NullTime) ([]DriverPerformanceRow, error)
	ReportDepartmentSummary(ctx context.Context, start, end sql.NullTime) ([]DepartmentSummaryRow, error)
	ReportResourceUsageRanged(ctx context.Context, start, end sql.NullTime) ([]ResourceUsageRangedRow, error)
	ReportDriverRatingsRanged(ctx context.Context, start, end sql.NullTime) ([]DriverRatingsRangedRow, error)
	ReportDriverActivityRanged(ctx context.Context, start, end sql.NullTime) ([]DriverActivityRangedRow, error)
	CheckVehicleSpdConflict(ctx context.Context, arg CheckVehicleSpdConflictParams) (int64, error)
	CheckDriverSpdConflict(ctx context.Context, arg CheckDriverSpdConflictParams) (int64, error)
	GetVehicleIDsWithActiveSpd(ctx context.Context) ([]int32, error)
	CheckMaintenanceConflict(ctx context.Context, arg CheckMaintenanceConflictParams) (int64, error)
	GetRoomIDByResourceID(ctx context.Context, resourceID int32) (int32, error)
	GetRoomKeeperIDByResourceID(ctx context.Context, resourceID int32) (sql.NullInt32, error)
	CreateRoomRating(ctx context.Context, arg CreateRoomRatingParams) (RoomRating, error)
	GetRoomRatingByBooking(ctx context.Context, bookingID int32) (RoomRating, error)
	GetRoomRatings(ctx context.Context, roomKeeperID int32) ([]GetRoomRatingsRow, error)
	SetRoomKeeper(ctx context.Context, roomID int32, roomKeeperID sql.NullInt32) (Room, error)
	GetRoomsByRoomKeeperID(ctx context.Context, roomKeeperID int32) ([]GetRoomsByRoomKeeperIDRow, error)
	ListRoomsWithKeeper(ctx context.Context) ([]RoomKeeperRoomRow, error)
	SetVehicleFixedDriver(ctx context.Context, arg SetVehicleFixedDriverParams) (Vehicle, error)
	ClearVehicleFixedDriverByDriver(ctx context.Context, driverID int32) error
	GetVehicleByFixedDriverID(ctx context.Context, driverID int32) (Vehicle, error)
	ListVehiclesWithFixedDriver(ctx context.Context) ([]FixedDriverVehicleRow, error)
	GetDriverIDsWithActiveSpd(ctx context.Context) ([]int32, error)
	GetPendingDriverRatings(ctx context.Context, userID int32) ([]PendingDriverRatingRow, error)

	// Penjaga data ganda (duplicate_guard.go)
	LockBookingCreate(ctx context.Context, userID int32) error
	FindActiveDuplicateBooking(ctx context.Context, userID, resourceID int32, start, end time.Time) (int32, error)
	FindRecentDuplicateFuel(ctx context.Context, vehicleID, fuelTypeID, odometer int32, quantity float64, within time.Duration) (int32, error)
}
