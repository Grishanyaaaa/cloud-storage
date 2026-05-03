package port

import (
	"context"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
)

// StorageUseCase is the input port for storage operations.
// Implemented by usecase.StorageService in the application layer.
type StorageUseCase interface {
	// Folders / nodes
	CreateFolder(ctx context.Context, actor *Actor, req dto.CreateFolderRequest) (*dto.NodeResponse, error)
	RenameNode(ctx context.Context, actor *Actor, nodeID string, req dto.RenameNodeRequest) (*dto.NodeResponse, error)
	MoveNode(ctx context.Context, actor *Actor, nodeID string, req dto.MoveNodeRequest) (*dto.NodeResponse, error)
	SoftDeleteNode(ctx context.Context, actor *Actor, nodeID string) error
	RestoreNode(ctx context.Context, actor *Actor, nodeID string) (*dto.NodeResponse, error)
	GetNode(ctx context.Context, actor *Actor, nodeID string) (*dto.NodeResponse, error)
	ListChildren(ctx context.Context, actor *Actor, req dto.ListChildrenRequest) (*dto.ListChildrenResponse, error)
	GetTree(ctx context.Context, actor *Actor, req dto.GetTreeRequest) (*dto.TreeNodeResponse, error)
	EnsureUserRoot(ctx context.Context, actor *Actor) (*dto.NodeResponse, error)

	// Files
	GenerateUploadURL(ctx context.Context, actor *Actor, req dto.GenerateUploadURLRequest) (*dto.UploadURLResponse, error)
	FinalizeUpload(ctx context.Context, actor *Actor, req dto.FinalizeUploadRequest) (*dto.NodeResponse, error)
	AbortUpload(ctx context.Context, actor *Actor, req dto.AbortUploadRequest) error
	GenerateDownloadURL(ctx context.Context, actor *Actor, req dto.GenerateDownloadURLRequest) (*dto.DownloadURLResponse, error)

	// Shares (owner-only)
	CreateShareLink(ctx context.Context, actor *Actor, req dto.CreateShareRequest) (*dto.ShareResponse, error)
	ListShareLinks(ctx context.Context, actor *Actor, req dto.ListSharesRequest) (*dto.ListSharesResponse, error)
	RevokeShareLink(ctx context.Context, actor *Actor, req dto.RevokeShareRequest) error

	// Public (share-token resolution by middleware)
	ResolveShareToken(ctx context.Context, rawToken string) (*Actor, error)

	// Janitor
	JanitorExpirePendingUploads(ctx context.Context) (int64, error)
	JanitorExpireShares(ctx context.Context) (int64, error)
}
