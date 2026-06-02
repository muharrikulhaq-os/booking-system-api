package repository

type AuditLogResponse struct {
	ID          int32   `json:"id"`
	UserID      *int32  `json:"user_id,omitempty"`
	Action      string  `json:"action"`
	EntityType  string  `json:"entity_type"`
	EntityID    *int32  `json:"entity_id,omitempty"`
	Description *string `json:"description,omitempty"`
	CreatedAt   string  `json:"created_at"`
	UserName    *string `json:"user_name,omitempty"`
}
