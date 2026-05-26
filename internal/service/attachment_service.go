package service

import (
	"context"
	"database/sql"
	"mime/multipart"
	"path/filepath"

	"booking-system-api/internal/repository"
	"booking-system-api/internal/util"
)

type AttachmentService struct {
	q *repository.Queries
}

func NewAttachmentService(db *sql.DB) *AttachmentService {
	return &AttachmentService{q: repository.New(db)}
}

func serializeAttachment(id int32, uploadedByID int32, uploaderName string,
	vehicleID, roomID, bookingID sql.NullInt32,
	filePath, fileName, fileType string,
	fileSize sql.NullInt32, description sql.NullString,
	createdAt any) map[string]any {
	return map[string]any{
		"id":           id,
		"uploadedById": uploadedByID,
		"uploaderName": uploaderName,
		"vehicleId":    nullInt32(vehicleID),
		"roomId":       nullInt32(roomID),
		"bookingId":    nullInt32(bookingID),
		"filePath":     filePath,
		"fileName":     fileName,
		"fileType":     fileType,
		"fileSize":     nullInt32(fileSize),
		"description":  nullStr(description),
		"createdAt":    createdAt,
	}
}

func (s *AttachmentService) ListByVehicle(ctx context.Context, vehicleID int32) ([]map[string]any, error) {
	rows, err := s.q.ListAttachmentsByVehicle(ctx, sql.NullInt32{Int32: vehicleID, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeAttachment(r.ID, r.UploadedById, r.UploaderName,
			r.VehicleId, r.RoomId, r.BookingId,
			r.FilePath, r.FileName, r.FileType, r.FileSize, r.Description, r.CreatedAt)
	}
	return out, nil
}

func (s *AttachmentService) ListByRoom(ctx context.Context, roomID int32) ([]map[string]any, error) {
	rows, err := s.q.ListAttachmentsByRoom(ctx, sql.NullInt32{Int32: roomID, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeAttachment(r.ID, r.UploadedById, r.UploaderName,
			r.VehicleId, r.RoomId, r.BookingId,
			r.FilePath, r.FileName, r.FileType, r.FileSize, r.Description, r.CreatedAt)
	}
	return out, nil
}

func (s *AttachmentService) ListByBooking(ctx context.Context, bookingID int32) ([]map[string]any, error) {
	rows, err := s.q.ListAttachmentsByBooking(ctx, sql.NullInt32{Int32: bookingID, Valid: true})
	if err != nil {
		return nil, err
	}
	out := make([]map[string]any, len(rows))
	for i, r := range rows {
		out[i] = serializeAttachment(r.ID, r.UploadedById, r.UploaderName,
			r.VehicleId, r.RoomId, r.BookingId,
			r.FilePath, r.FileName, r.FileType, r.FileSize, r.Description, r.CreatedAt)
	}
	return out, nil
}

func (s *AttachmentService) UploadForVehicle(ctx context.Context, vehicleID, uploaderID int32, fh *multipart.FileHeader, description string) (map[string]any, error) {
	filePath, err := util.SaveUploadedFile(fh)
	if err != nil {
		return nil, util.NewError(400, err.Error(), util.ErrBadRequest)
	}

	a, err := s.q.CreateAttachmentForVehicle(ctx, repository.CreateAttachmentForVehicleParams{
		UploadedById: uploaderID,
		VehicleId:    sql.NullInt32{Int32: vehicleID, Valid: true},
		FilePath:     filePath,
		FileName:     filepath.Base(fh.Filename),
		FileType:     util.GetFileMimeType(fh),
		FileSize:     sql.NullInt32{Int32: int32(fh.Size), Valid: true},
		Description:  sql.NullString{String: description, Valid: description != ""},
	})
	if err != nil {
		util.DeleteUploadedFile(filePath)
		return nil, err
	}

	row, _ := s.q.GetAttachmentByID(ctx, a.ID)
	return serializeAttachment(row.ID, row.UploadedById, row.UploaderName,
		row.VehicleId, row.RoomId, row.BookingId,
		row.FilePath, row.FileName, row.FileType, row.FileSize, row.Description, row.CreatedAt), nil
}

func (s *AttachmentService) UploadForRoom(ctx context.Context, roomID, uploaderID int32, fh *multipart.FileHeader, description string) (map[string]any, error) {
	filePath, err := util.SaveUploadedFile(fh)
	if err != nil {
		return nil, util.NewError(400, err.Error(), util.ErrBadRequest)
	}

	a, err := s.q.CreateAttachmentForRoom(ctx, repository.CreateAttachmentForRoomParams{
		UploadedById: uploaderID,
		RoomId:       sql.NullInt32{Int32: roomID, Valid: true},
		FilePath:     filePath,
		FileName:     filepath.Base(fh.Filename),
		FileType:     util.GetFileMimeType(fh),
		FileSize:     sql.NullInt32{Int32: int32(fh.Size), Valid: true},
		Description:  sql.NullString{String: description, Valid: description != ""},
	})
	if err != nil {
		util.DeleteUploadedFile(filePath)
		return nil, err
	}

	row, _ := s.q.GetAttachmentByID(ctx, a.ID)
	return serializeAttachment(row.ID, row.UploadedById, row.UploaderName,
		row.VehicleId, row.RoomId, row.BookingId,
		row.FilePath, row.FileName, row.FileType, row.FileSize, row.Description, row.CreatedAt), nil
}

func (s *AttachmentService) UploadForBooking(ctx context.Context, bookingID, uploaderID int32, fh *multipart.FileHeader, description string) (map[string]any, error) {
	filePath, err := util.SaveUploadedFile(fh)
	if err != nil {
		return nil, util.NewError(400, err.Error(), util.ErrBadRequest)
	}

	a, err := s.q.CreateAttachmentForBooking(ctx, repository.CreateAttachmentForBookingParams{
		UploadedById: uploaderID,
		BookingId:    sql.NullInt32{Int32: bookingID, Valid: true},
		FilePath:     filePath,
		FileName:     filepath.Base(fh.Filename),
		FileType:     util.GetFileMimeType(fh),
		FileSize:     sql.NullInt32{Int32: int32(fh.Size), Valid: true},
		Description:  sql.NullString{String: description, Valid: description != ""},
	})
	if err != nil {
		util.DeleteUploadedFile(filePath)
		return nil, err
	}

	row, _ := s.q.GetAttachmentByID(ctx, a.ID)
	return serializeAttachment(row.ID, row.UploadedById, row.UploaderName,
		row.VehicleId, row.RoomId, row.BookingId,
		row.FilePath, row.FileName, row.FileType, row.FileSize, row.Description, row.CreatedAt), nil
}

func (s *AttachmentService) Delete(ctx context.Context, id int32) error {
	att, err := s.q.GetAttachmentByID(ctx, id)
	if err != nil {
		return util.ErrNotFound
	}
	if err = s.q.DeleteAttachment(ctx, id); err != nil {
		return err
	}
	util.DeleteUploadedFile(att.FilePath)
	return nil
}
