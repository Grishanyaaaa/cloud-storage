package ttl

import (
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/infrastructure/config"
)

// Compile-time check
var _ port.TTLPolicy = (*Policy)(nil)

// Policy is the production size-based TTL policy described in blueprint §13.1.
type Policy struct {
	cfg config.TTLConfig
}

func NewPolicy(cfg config.TTLConfig) *Policy { return &Policy{cfg: cfg} }

func (p *Policy) UploadTTL(size valueobject.SizeBytes) time.Duration {
	mib := size.MiB()
	d := p.cfg.BaseUpload + time.Duration(mib)*p.cfg.PerMiBUpload
	return clamp(d, p.cfg.MinUpload, p.cfg.MaxUpload)
}

func (p *Policy) DownloadTTL(size valueobject.SizeBytes) time.Duration {
	mib := size.MiB()
	d := p.cfg.BaseDownload + time.Duration(mib)*p.cfg.PerMiBDownload
	return clamp(d, p.cfg.MinDownload, p.cfg.MaxDownload)
}

func clamp(d, lo, hi time.Duration) time.Duration {
	if d < lo {
		return lo
	}
	if d > hi {
		return hi
	}
	return d
}
