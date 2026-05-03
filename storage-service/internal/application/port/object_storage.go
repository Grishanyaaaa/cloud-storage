package port

import (
	"context"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// ObjectMetadata returned by HeadObject after a client uploaded directly to S3.
type ObjectMetadata struct {
	SizeBytes int64
	ETag      string
	MimeType  string
}

// PresignedURL describes a single pre-signed URL returned to the client.
// Headers are extra HTTP headers the client MUST include verbatim when uploading
// (e.g. Content-Length, Content-Type) for the signature to be valid.
type PresignedURL struct {
	URL       string
	Method    string
	Headers   map[string]string
	ExpiresAt time.Time
}

// PresignUploadInput are the inputs for a pre-signed PUT.
type PresignUploadInput struct {
	Key       valueobject.StorageKey
	SizeBytes int64
	MimeType  valueobject.MimeType
	TTL       time.Duration
}

// PresignDownloadInput are the inputs for a pre-signed GET.
type PresignDownloadInput struct {
	Key                valueobject.StorageKey
	TTL                time.Duration
	Filename           string // for Content-Disposition
	Disposition        string // "inline" | "attachment"
	ResponseMimeType   string
}

// ObjectStorage abstracts the S3-compatible object store.
// Storage service only touches metadata; the binary payload always travels
// directly between client and S3 via pre-signed URLs.
type ObjectStorage interface {
	// PresignUpload generates a single pre-signed PUT URL.
	PresignUpload(ctx context.Context, in PresignUploadInput) (*PresignedURL, error)

	// PresignDownload generates a single pre-signed GET URL.
	PresignDownload(ctx context.Context, in PresignDownloadInput) (*PresignedURL, error)

	// HeadObject returns metadata about an uploaded object.
	// Returns nil, nil when the object does not exist.
	HeadObject(ctx context.Context, key valueobject.StorageKey) (*ObjectMetadata, error)

	// DeleteObject removes an object from the bucket. Idempotent (no-error on missing).
	DeleteObject(ctx context.Context, key valueobject.StorageKey) error
}
