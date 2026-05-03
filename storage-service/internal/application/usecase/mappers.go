package usecase

import (
	"net/url"
	"strings"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
)

// toNodeResponse converts a Node entity (and optional FileBlob) into the HTTP response DTO.
// blob may be nil for folder nodes or when the file blob is not yet attached.
func toNodeResponse(n *entity.Node, blob *entity.FileBlob) *dto.NodeResponse {
	resp := &dto.NodeResponse{
		ID:        n.ID().String(),
		OwnerID:   n.OwnerID().String(),
		Kind:      n.Kind().String(),
		Name:      n.Name().String(),
		Path:      n.Path().String(),
		Depth:     n.Depth(),
		CreatedAt: n.CreatedAt(),
		UpdatedAt: n.UpdatedAt(),
		DeletedAt: n.DeletedAt(),
	}
	if pid := n.ParentID(); pid != nil {
		s := pid.String()
		resp.ParentID = &s
	}
	if blob != nil {
		size := blob.Size().Value()
		mime := blob.MimeType().String()
		status := string(blob.Status())
		resp.SizeBytes = &size
		resp.MimeType = &mime
		resp.Status = &status
	}
	return resp
}

// joinURL joins a base URL with a path-suffix, taking care of stray slashes.
func joinURL(base, suffix string) string {
	if base == "" {
		return suffix
	}
	base = strings.TrimRight(base, "/")
	suffix = strings.TrimLeft(suffix, "/")
	return base + "/" + suffix
}

// safeFilename ensures a filename is safe to put into a Content-Disposition header.
func safeFilename(name string) string {
	return url.PathEscape(name)
}
