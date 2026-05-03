package port

import (
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// TTLPolicy decides how long pre-signed URLs are valid.
// The policy is size-based (см. blueprint §13.1):
//
//	upload TTL   = clamp(BaseUpload + sizeMiB * PerMiBUpload, MinUpload, MaxUpload)
//	download TTL = clamp(BaseDownload + sizeMiB * PerMiBDownload, MinDownload, MaxDownload)
type TTLPolicy interface {
	UploadTTL(size valueobject.SizeBytes) time.Duration
	DownloadTTL(size valueobject.SizeBytes) time.Duration
}
