package usecase

import (
	"log/slog"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/repository"
)

// Compile-time check: StorageService implements port.StorageUseCase
var _ port.StorageUseCase = (*StorageService)(nil)

// StorageService implements the StorageUseCase interface.
// It coordinates domain entities, repositories and infrastructure adapters.
type StorageService struct {
	nodeRepo     repository.NodeRepository
	blobRepo     repository.FileBlobRepository
	rootRepo     repository.UserRootRepository
	shareRepo    repository.ShareRepository
	txManager    repository.TransactionManager
	storage      port.ObjectStorage
	ttl          port.TTLPolicy
	ids          port.IDGenerator
	tokens       port.TokenGenerator
	policy       *authorizationPolicy
	publicBaseURL string
	maxFileSize   int64
	logger       *slog.Logger
}

// NewStorageService creates a new StorageService.
func NewStorageService(
	nodeRepo repository.NodeRepository,
	blobRepo repository.FileBlobRepository,
	rootRepo repository.UserRootRepository,
	shareRepo repository.ShareRepository,
	txManager repository.TransactionManager,
	storage port.ObjectStorage,
	ttl port.TTLPolicy,
	ids port.IDGenerator,
	tokens port.TokenGenerator,
	publicBaseURL string,
	maxFileSize int64,
	logger *slog.Logger,
) *StorageService {
	return &StorageService{
		nodeRepo:      nodeRepo,
		blobRepo:      blobRepo,
		rootRepo:      rootRepo,
		shareRepo:     shareRepo,
		txManager:     txManager,
		storage:       storage,
		ttl:           ttl,
		ids:           ids,
		tokens:        tokens,
		policy:        newAuthorizationPolicy(),
		publicBaseURL: publicBaseURL,
		maxFileSize:   maxFileSize,
		logger:        logger,
	}
}
