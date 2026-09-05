package audit

import (
	"context"
	"time"
)

// Standard action set. More can be added as the system grows.
const (
	ActionLogin   = "LOGIN"
	ActionLogout  = "LOGOUT"
	ActionCreate  = "CREATE"
	ActionUpdate  = "UPDATE"
	ActionDelete  = "DELETE"
	ActionRestore = "RESTORE"
	ActionApprove = "APPROVE"
	ActionReject  = "REJECT"
)

const (
	ActionRoleAssigned      = "ROLE_ASSIGNED"
	ActionRoleRemoved       = "ROLE_REMOVED"
	ActionPermissionGranted = "PERMISSION_GRANTED"
	ActionPermissionRevoked = "PERMISSION_REVOKED"
	ActionRateLimitCreated  = "RATE_LIMIT_CREATED"
	ActionRateLimitUpdated  = "RATE_LIMIT_UPDATED"
	ActionRateLimitDeleted  = "RATE_LIMIT_DELETED"
)

type Entry struct {
	UserID     int64          `json:"user_id"`
	Action     string         `json:"action"`
	Resource   string         `json:"resource"`
	ResourceID string         `json:"resource_id"`
	Method     string         `json:"method,omitempty"`
	Path       string         `json:"path,omitempty"`
	IPAddress  string         `json:"ip_address,omitempty"`
	UserAgent  string         `json:"user_agent,omitempty"`
	RequestID  string         `json:"request_id,omitempty"`
	OldData    map[string]any `json:"old_data,omitempty"`
	NewData    map[string]any `json:"new_data,omitempty"`
	Metadata   map[string]any `json:"metadata,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
}

type Logger interface {
	Log(ctx context.Context, e Entry) error
}

type Repository interface {
	Insert(ctx context.Context, e Entry) error
	Search(ctx context.Context, filter SearchFilter) ([]Entry, int64, error)
}

type SearchFilter struct {
	UserID     int64
	Action     string
	Resource   string
	ResourceID string
	RequestID  string
	Page       int
	PageSize   int
}
