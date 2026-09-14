package repository

import "time"

type AuditLogResponse struct {
	ID          int32     `json:"id"`
	UserID      *int32    `json:"userId,omitempty"`
	Action      string    `json:"action"`
	EntityType  string    `json:"entityType"`
	EntityID    *int32    `json:"entityId,omitempty"`
	Description *string   `json:"description,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	IPAddress   *string   `json:"ipAddress,omitempty"`
	UserAgent   *string   `json:"userAgent,omitempty"`
	UserName    *string   `json:"userName,omitempty"`
}
