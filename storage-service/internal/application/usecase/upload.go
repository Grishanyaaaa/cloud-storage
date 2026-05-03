package usecase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/repository"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// GenerateUploadURL creates a pending file node + blob and returns a pre-signed PUT URL.
// Owner-only: writing into someone else's tree is never allowed via share-links.
func (s *StorageService) GenerateUploadURL(ctx context.Context, actor *port.Actor, req dto.GenerateUploadURLRequest) (*dto.UploadURLResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := s.policy.allowOwner(actor, nil); err != nil {
		return nil, err
	}
	if s.maxFileSize > 0 && req.SizeBytes > s.maxFileSize {
		return nil, domainerr.ErrFileTooLarge
	}

	parentID, err := valueobject.ParseNodeID(req.ParentID)
	if err != nil {
		return nil, err
	}
	name, err := valueobject.NewNodeName(req.Name)
	if err != nil {
		return nil, err
	}
	mime, err := valueobject.NewMimeType(req.MimeType)
	if err != nil {
		return nil, err
	}
	size, err := valueobject.NewSizeBytes(req.SizeBytes)
	if err != nil {
		return nil, err
	}

	parent, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, parentID)
	if err != nil {
		return nil, err
	}
	if !parent.IsFolder() {
		return nil, domainerr.ErrNodeKindMismatch
	}
	if parent.IsDeleted() {
		return nil, domainerr.ErrNodeAlreadyDeleted
	}

	id := s.ids.NewNodeID()
	now := time.Now()
	fileNode, err := entity.NewChildNode(id, actor.UserID, parent, valueobject.KindFile, name, now)
	if err != nil {
		return nil, err
	}
	storageKey, err := valueobject.NewStorageKey(actor.UserID, id)
	if err != nil {
		return nil, err
	}
	ttl := s.ttl.UploadTTL(size)
	expiresAt := now.Add(ttl)
	blob := entity.NewPendingFileBlob(id, storageKey, mime, expiresAt, now)

	txErr := s.txManager.WithTransaction(ctx, func(ctx context.Context, tx repository.Transaction) error {
		if err := s.nodeRepo.CreateTx(ctx, tx, fileNode); err != nil {
			return err
		}
		if err := s.blobRepo.CreateTx(ctx, tx, blob); err != nil {
			return err
		}
		return nil
	})
	if txErr != nil {
		if errors.Is(txErr, domainerr.ErrNodeNameTaken) {
			return nil, txErr
		}
		return nil, fmt.Errorf("create pending upload: %w", txErr)
	}

	pre, err := s.storage.PresignUpload(ctx, port.PresignUploadInput{
		Key:       storageKey,
		SizeBytes: req.SizeBytes,
		MimeType:  mime,
		TTL:       ttl,
	})
	if err != nil {
		return nil, fmt.Errorf("presign upload: %w", err)
	}

	return &dto.UploadURLResponse{
		NodeID:    id.String(),
		URL:       pre.URL,
		Method:    pre.Method,
		Headers:   pre.Headers,
		ExpiresAt: pre.ExpiresAt,
	}, nil
}

// FinalizeUpload activates a pending blob after the client uploaded successfully.
// Owner-only.
func (s *StorageService) FinalizeUpload(ctx context.Context, actor *port.Actor, req dto.FinalizeUploadRequest) (*dto.NodeResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}
	if err := s.policy.allowOwner(actor, nil); err != nil {
		return nil, err
	}
	id, err := valueobject.ParseNodeID(req.NodeID)
	if err != nil {
		return nil, err
	}
	node, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, id)
	if err != nil {
		return nil, err
	}
	if !node.IsFile() {
		return nil, domainerr.ErrNodeKindMismatch
	}

	blob, err := s.blobRepo.GetByNodeID(ctx, node.ID())
	if err != nil {
		return nil, err
	}
	if !blob.IsPending() {
		return nil, domainerr.ErrFileNotPending
	}

	// Verify the object exists and observe its size on S3 to defend against
	// clients that lie about the uploaded size.
	meta, err := s.storage.HeadObject(ctx, blob.StorageKey())
	if err != nil {
		return nil, fmt.Errorf("head object: %w", err)
	}
	if meta == nil {
		return nil, domainerr.ErrFileBlobNotFound
	}
	if meta.SizeBytes != req.SizeBytes {
		return nil, domainerr.ErrInvalidSize
	}

	size, err := valueobject.NewSizeBytes(meta.SizeBytes)
	if err != nil {
		return nil, err
	}
	if err := blob.Activate(size, req.Checksum, time.Now()); err != nil {
		return nil, err
	}
	if err := s.blobRepo.Update(ctx, blob); err != nil {
		return nil, fmt.Errorf("activate blob: %w", err)
	}
	return toNodeResponse(node, blob), nil
}

// AbortUpload marks a pending blob as failed and best-effort deletes the object.
// Owner-only.
func (s *StorageService) AbortUpload(ctx context.Context, actor *port.Actor, req dto.AbortUploadRequest) error {
	if err := s.policy.allowOwner(actor, nil); err != nil {
		return err
	}
	if req.NodeID == "" {
		return domainerr.ErrInvalidNodeID
	}
	id, err := valueobject.ParseNodeID(req.NodeID)
	if err != nil {
		return err
	}
	node, err := s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, id)
	if err != nil {
		return err
	}
	if !node.IsFile() {
		return domainerr.ErrNodeKindMismatch
	}
	blob, err := s.blobRepo.GetByNodeID(ctx, node.ID())
	if err != nil {
		return err
	}
	if !blob.IsPending() {
		return domainerr.ErrFileNotPending
	}
	if err := blob.Fail(time.Now()); err != nil {
		return err
	}
	if err := s.blobRepo.Update(ctx, blob); err != nil {
		return fmt.Errorf("fail blob: %w", err)
	}
	// Best-effort delete; ignore not-found.
	if delErr := s.storage.DeleteObject(ctx, blob.StorageKey()); delErr != nil {
		s.logger.Warn("delete aborted upload object", "key", blob.StorageKey().String(), "err", delErr.Error())
	}
	return nil
}
