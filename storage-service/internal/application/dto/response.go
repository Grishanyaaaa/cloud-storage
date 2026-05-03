package dto

import "time"

// NodeResponse is the canonical view of a node returned over HTTP.
type NodeResponse struct {
	ID        string     `json:"id"`
	OwnerID   string     `json:"owner_id"`
	ParentID  *string    `json:"parent_id,omitempty"`
	Kind      string     `json:"kind"`
	Name      string     `json:"name"`
	Path      string     `json:"path"`
	Depth     int        `json:"depth"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
	DeletedAt *time.Time `json:"deleted_at,omitempty"`

	// File-only (omitted for folders)
	SizeBytes *int64  `json:"size_bytes,omitempty"`
	MimeType  *string `json:"mime_type,omitempty"`
	Status    *string `json:"status,omitempty"` // pending | active | failed
}

// ListChildrenResponse is the paginated listing of a folder.
type ListChildrenResponse struct {
	Items      []NodeResponse `json:"items"`
	NextCursor string         `json:"next_cursor,omitempty"`
}

// TreeNodeResponse is a single node in a partial tree view.
type TreeNodeResponse struct {
	NodeResponse
	Children []TreeNodeResponse `json:"children,omitempty"`
}

// UploadURLResponse contains the pre-signed PUT URL the client uses to upload directly to S3.
type UploadURLResponse struct {
	NodeID    string            `json:"node_id"`
	URL       string            `json:"url"`
	Method    string            `json:"method"` // "PUT"
	Headers   map[string]string `json:"headers,omitempty"`
	ExpiresAt time.Time         `json:"expires_at"`
}

// DownloadURLResponse contains the pre-signed GET URL.
type DownloadURLResponse struct {
	URL       string    `json:"url"`
	Method    string    `json:"method"` // "GET"
	ExpiresAt time.Time `json:"expires_at"`
}

// ShareResponse describes a single share-link as returned to the owner.
// The raw token is included only for the response of CreateShareLink.
type ShareResponse struct {
	ID         string     `json:"id"`
	NodeID     string     `json:"node_id"`
	Permission string     `json:"permission"`
	URL        string     `json:"url,omitempty"`   // owner-only convenience
	Token      string     `json:"token,omitempty"` // ONLY present on Create response
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	RevokedAt  *time.Time `json:"revoked_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ListSharesResponse is the listing of shares for a node.
type ListSharesResponse struct {
	Items []ShareResponse `json:"items"`
}

// PublicShareResponse describes a share to a public consumer (no owner data leaked).
type PublicShareResponse struct {
	NodeID     string     `json:"node_id"`
	Kind       string     `json:"kind"`
	Name       string     `json:"name"`
	Permission string     `json:"permission"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
}
