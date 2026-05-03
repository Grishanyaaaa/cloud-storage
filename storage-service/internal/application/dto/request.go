package dto

import (
	"strings"
)

// CreateFolderRequest creates a folder under ParentID.
type CreateFolderRequest struct {
	ParentID string `json:"parent_id"`
	Name     string `json:"name"`
}

func (r *CreateFolderRequest) Validate() error {
	if r.ParentID == "" {
		return ErrParentIDRequired
	}
	if strings.TrimSpace(r.Name) == "" {
		return ErrNameRequired
	}
	return nil
}

// RenameNodeRequest renames an existing node (taken from URL path).
type RenameNodeRequest struct {
	Name string `json:"name"`
}

func (r *RenameNodeRequest) Validate() error {
	if strings.TrimSpace(r.Name) == "" {
		return ErrNameRequired
	}
	return nil
}

// MoveNodeRequest moves a node to a new parent.
type MoveNodeRequest struct {
	NewParentID string `json:"new_parent_id"`
}

func (r *MoveNodeRequest) Validate() error {
	if r.NewParentID == "" {
		return ErrParentIDRequired
	}
	return nil
}

// ListChildrenRequest paginates listing of folder children.
type ListChildrenRequest struct {
	ParentID       string
	Cursor         string
	Limit          int
	IncludeDeleted bool
}

func (r *ListChildrenRequest) Validate() error {
	if r.ParentID == "" {
		return ErrParentIDRequired
	}
	if r.Limit < 0 || r.Limit > 1000 {
		return ErrInvalidLimit
	}
	if r.Limit == 0 {
		r.Limit = 100
	}
	return nil
}

// GetTreeRequest requests a partial tree view starting at RootID (or owner's root if empty).
type GetTreeRequest struct {
	RootID         string
	MaxDepth       int
	IncludeDeleted bool
}

// GenerateUploadURLRequest requests a pre-signed PUT URL for uploading a new file.
type GenerateUploadURLRequest struct {
	ParentID  string `json:"parent_id"`
	Name      string `json:"name"`
	SizeBytes int64  `json:"size_bytes"`
	MimeType  string `json:"mime_type"`
}

func (r *GenerateUploadURLRequest) Validate() error {
	if r.ParentID == "" {
		return ErrParentIDRequired
	}
	if strings.TrimSpace(r.Name) == "" {
		return ErrNameRequired
	}
	if r.SizeBytes <= 0 {
		return ErrSizeRequired
	}
	return nil
}

// FinalizeUploadRequest activates a previously generated pre-signed PUT.
type FinalizeUploadRequest struct {
	NodeID    string `json:"-"`
	SizeBytes int64  `json:"size_bytes"`
	Checksum  string `json:"checksum"` // sha256 hex
}

func (r *FinalizeUploadRequest) Validate() error {
	if r.NodeID == "" {
		return ErrNodeIDRequired
	}
	if r.SizeBytes <= 0 {
		return ErrSizeRequired
	}
	if r.Checksum == "" {
		return ErrEmptyChecksum
	}
	if len(r.Checksum) != 64 {
		return ErrInvalidChecksum
	}
	for _, c := range r.Checksum {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			return ErrInvalidChecksum
		}
	}
	return nil
}

// AbortUploadRequest aborts a pending upload.
type AbortUploadRequest struct {
	NodeID string `json:"-"`
}

// GenerateDownloadURLRequest requests a pre-signed GET URL.
type GenerateDownloadURLRequest struct {
	NodeID      string
	Disposition string // inline | attachment (default: attachment)
}

func (r *GenerateDownloadURLRequest) Validate() error {
	if r.NodeID == "" {
		return ErrNodeIDRequired
	}
	switch r.Disposition {
	case "", "inline", "attachment":
		return nil
	default:
		return ErrInvalidDisposition
	}
}

// CreateShareRequest creates a share-link for a node.
type CreateShareRequest struct {
	NodeID     string `json:"-"`
	Permission string `json:"permission"`
	ExpiresIn  string `json:"expires_in"` // optional, e.g. "24h"; empty = no expiration
}

func (r *CreateShareRequest) Validate() error {
	if r.NodeID == "" {
		return ErrNodeIDRequired
	}
	switch r.Permission {
	case "view", "edit":
		// ok
	default:
		return ErrInvalidPermission
	}
	return nil
}

// ListSharesRequest lists shares for a node.
type ListSharesRequest struct {
	NodeID         string
	IncludeRevoked bool
}

// RevokeShareRequest revokes a share by ID.
type RevokeShareRequest struct {
	ShareID string
}
