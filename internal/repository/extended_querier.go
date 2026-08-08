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
	CreateReturnReport(ctx context.Context, bookingID, submittedByID int32, note, location string, odometer sql.NullInt32) (BookingReturnReport, error)
	GetReturnReport(ctx context.Context, bookingID int32) (BookingReturnReportRow, error)
	SetBookingStartTrip(ctx context.Context, bookingID int32, odometer sql.NullInt32, location, photoURL sql.NullString) error
	GetRoomKeeperByUserID(ctx context.Context, userID int32) (RoomKeeper, error)
	GetBookingActivity(ctx context.Context, bookingID int32) ([]BookingActivityRow, error)
	ReportOverview(ctx context.Context, start, end time.Time) (OverviewRow, error)
	ReportBookingTrend(ctx context.Context, groupBy string, periods int) ([]BookingTrendRow, error)
	ReportBookingsByDepartment(ctx context.Context, start, end sql.NullTime) ([]BookingByDepartmentRow, error)
	ReportBookingsByResource(ctx context.Context, start, end sql.NullTime) ([]BookingByResourceRow, error)
	ReportApprovalPerformance(ctx context.Context, start, end sql.NullTime) (ApprovalPerformanceRow, error)
	ReportCostSummary(ctx context.Context, start, end sql.NullTime) (CostSummaryRow, error)
	ReportCostByVehicle(ctx context.Context, start, end sql.NullTime) ([]CostByVehicleRow, error)
	ReportCostByDepartment(ctx context.Context, start, end sql.NullTime) ([]CostByDepartmentRow, error)
	ReportCostTrend(ctx context.Context, groupBy string, periods int) ([]CostTrendRow, error)
	ReportDriverPerformance(ctx context.Context, start, end sql.NullTime) ([]DriverPerformanceRow, error)
	ReportDepartmentSummary(ctx context.Context, start, end sql.NullTime) ([]DepartmentSummaryRow, error)
}
