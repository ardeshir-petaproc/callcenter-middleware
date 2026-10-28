package domain

import "time"

// Knowledge source kinds, statuses and visibility levels.
const (
	KBFile  = "file"
	KBText  = "text"
	KBEmail = "email"

	KBActive   = "active"
	KBDraft    = "draft"
	KBArchived = "archived"

	KBVisibilityAll        = "all"        // every employee of the tenant
	KBVisibilityRestricted = "restricted" // kb.restricted permission or an allowed role only
)

// KnowledgeSource is a document, note or email the business adds to its
// knowledge base. Scope fields attach it to parts of the organization; the
// AI may only use sources that are active, valid, AI-enabled and in scope.
type KnowledgeSource struct {
	ID            int64    `json:"id"`
	TenantID      int64    `json:"tenant_id"`
	Title         string   `json:"title"`
	Kind          string   `json:"kind"`
	Category      string   `json:"category"`
	Tags          []string `json:"tags"`
	DepartmentIDs []int64  `json:"department_ids"`
	QueueIDs      []int64  `json:"queue_ids"`
	JobProfileIDs []int64  `json:"job_profile_ids"`
	Topics        []string `json:"topics"`
	Visibility    string   `json:"visibility"`
	AllowedRoles  []string `json:"allowed_roles"` // admin, supervisor, agent or role:<custom id>
	AIEnabled     bool     `json:"ai_enabled"`
	Status        string   `json:"status"`
	ValidUntil    string   `json:"valid_until"` // YYYY-MM-DD, empty = no expiry
	OwnerID       int64    `json:"owner_id"`
	Version       int      `json:"version"`
	UpdatedAt     string   `json:"updated_at"`
	FileName      string   `json:"file_name"`
	MimeType      string   `json:"mime_type"`
	Size          int64    `json:"size"`
	StorageKey    string   `json:"storage_key,omitempty"`
	Checksum      string   `json:"checksum"`
	TextChars     int      `json:"text_chars"`
	ConnectorID   int64    `json:"connector_id"`
	EmailID       int64    `json:"email_id"`
}

// Expired reports whether the validity date has passed.
func (k *KnowledgeSource) Expired(now time.Time) bool {
	if k.ValidUntil == "" {
		return false
	}
	d, err := time.Parse("2006-01-02", k.ValidUntil)
	return err == nil && now.After(d.Add(24*time.Hour))
}

// KBHit is one search result.
type KBHit struct {
	SourceID int64   `json:"source_id"`
	Title    string  `json:"title"`
	Category string  `json:"category"`
	Chunk    int     `json:"chunk"`
	Snippet  string  `json:"snippet"`
	Text     string  `json:"text,omitempty"`
	Score    float64 `json:"score"`
}

// KBAccess describes who is searching, for permission filtering.
type KBAccess struct {
	AI            bool
	Role          Role
	CustomRoleID  int64
	CanRestricted bool
	Manager       bool // kb.manage sees everything of the tenant
	DepartmentIDs []int64
	Categories    []string // AI profile knowledge scope (empty = all)
}
