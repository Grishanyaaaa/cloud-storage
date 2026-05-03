package entity

import (
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// BlobStatus is the lifecycle state of a file blob.
type BlobStatus string

const (
	BlobPending BlobStatus = "pending"
	BlobActive  BlobStatus = "active"
	BlobFailed  BlobStatus = "failed"
)

// FileBlob is the binary part of a file node.
// Lifecycle: pending → active (FinalizeUpload) | failed (AbortUpload).
// Size and checksum are populated only on activation.
type FileBlob struct {
	nodeID     valueobject.NodeID
	storageKey valueobject.StorageKey
	mime       valueobject.MimeType
	size       valueobject.SizeBytes
	checksum   string // optional sha256 (hex), provided on FinalizeUpload
	status     BlobStatus
	createdAt  time.Time
	updatedAt  time.Time
	expiresAt  *time.Time // for pending blobs only — janitor uses this to mark failed
}

// NewPendingFileBlob creates a fresh pending blob bound to nodeID.
func NewPendingFileBlob(
	nodeID valueobject.NodeID,
	key valueobject.StorageKey,
	mime valueobject.MimeType,
	expiresAt time.Time,
	now time.Time,
) *FileBlob {
	return &FileBlob{
		nodeID:     nodeID,
		storageKey: key,
		mime:       mime,
		size:       valueobject.SizeBytes{},
		checksum:   "",
		status:     BlobPending,
		createdAt:  now,
		updatedAt:  now,
		expiresAt:  &expiresAt,
	}
}

// ReconstructFileBlob reconstructs a FileBlob from persistence.
func ReconstructFileBlob(
	nodeID valueobject.NodeID,
	storageKey valueobject.StorageKey,
	mime valueobject.MimeType,
	size valueobject.SizeBytes,
	checksum string,
	status BlobStatus,
	createdAt time.Time,
	updatedAt time.Time,
	expiresAt *time.Time,
) *FileBlob {
	return &FileBlob{
		nodeID:     nodeID,
		storageKey: storageKey,
		mime:       mime,
		size:       size,
		checksum:   checksum,
		status:     status,
		createdAt:  createdAt,
		updatedAt:  updatedAt,
		expiresAt:  expiresAt,
	}
}

// Getters
func (b *FileBlob) NodeID() valueobject.NodeID         { return b.nodeID }
func (b *FileBlob) StorageKey() valueobject.StorageKey { return b.storageKey }
func (b *FileBlob) MimeType() valueobject.MimeType     { return b.mime }
func (b *FileBlob) Size() valueobject.SizeBytes        { return b.size }
func (b *FileBlob) Checksum() string                   { return b.checksum }
func (b *FileBlob) Status() BlobStatus                 { return b.status }
func (b *FileBlob) CreatedAt() time.Time               { return b.createdAt }
func (b *FileBlob) UpdatedAt() time.Time               { return b.updatedAt }
func (b *FileBlob) ExpiresAt() *time.Time              { return b.expiresAt }
func (b *FileBlob) IsPending() bool                    { return b.status == BlobPending }
func (b *FileBlob) IsActive() bool                     { return b.status == BlobActive }
func (b *FileBlob) IsFailed() bool                     { return b.status == BlobFailed }

// Activate transitions pending → active and stores observed size/checksum.
func (b *FileBlob) Activate(size valueobject.SizeBytes, checksum string, now time.Time) error {
	if b.status != BlobPending {
		return domainerr.ErrFileNotPending
	}
	b.size = size
	b.checksum = checksum
	b.status = BlobActive
	b.updatedAt = now
	b.expiresAt = nil
	return nil
}

// Fail transitions pending → failed.
func (b *FileBlob) Fail(now time.Time) error {
	if b.status != BlobPending {
		return domainerr.ErrFileNotPending
	}
	b.status = BlobFailed
	b.updatedAt = now
	b.expiresAt = nil
	return nil
}
