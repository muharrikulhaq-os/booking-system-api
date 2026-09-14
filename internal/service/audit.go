package service

import (
	"context"
	"database/sql"

	"booking-system-api/internal/repository"
)

// AuditActor carries request-scoped metadata (who performed the action and
// from where) needed to write an audit log entry. Handlers build this once
// via the delivery/http `auditActor(c)` helper and pass it into service
// methods that mutate state, instead of threading userID/ip/userAgent as
// three separate params through every Create/Update/Delete/ToggleActive.
type AuditActor struct {
	UserID    int32
	IP        string
	UserAgent string
}

// logAudit writes one audit_logs row. Best-effort and fire-and-forget, same
// as every pre-existing CreateAuditLog call site in this codebase - a
// failure to write the audit trail must never fail the actual request.
func logAudit(ctx context.Context, q repository.ExtendedQuerier, actor AuditActor, action, entityType string, entityID int32, description string) {
	_, _ = q.CreateAuditLog(ctx, repository.CreateAuditLogParams{
		UserId:      sql.NullInt32{Int32: actor.UserID, Valid: actor.UserID != 0},
		Action:      action,
		EntityType:  entityType,
		EntityId:    sql.NullInt32{Int32: entityID, Valid: true},
		Description: sql.NullString{String: description, Valid: description != ""},
		IpAddress:   sql.NullString{String: actor.IP, Valid: actor.IP != ""},
		UserAgent:   sql.NullString{String: actor.UserAgent, Valid: actor.UserAgent != ""},
	})
}
