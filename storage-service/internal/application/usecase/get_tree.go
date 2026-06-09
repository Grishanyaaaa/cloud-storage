package usecase

import (
	"context"
	"fmt"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

const defaultMaxDepth = 5

// GetTree returns a partial tree view. Allowed for owner OR share-link.
// Without RootID, owners get their full root subtree; share-link consumers
// always get the share root subtree (RootID is ignored).
func (s *StorageService) GetTree(ctx context.Context, actor *port.Actor, req dto.GetTreeRequest) (*dto.TreeNodeResponse, error) {
	if actor == nil {
		return nil, domainerr.ErrForbidden
	}

	var root *entity.Node
	var err error

	switch {
	case actor.IsShareLink():
		root = actor.ShareRoot
	case req.RootID != "":
		id, perr := valueobject.ParseNodeID(req.RootID)
		if perr != nil {
			return nil, perr
		}
		root, err = s.nodeRepo.GetByIDForOwner(ctx, actor.UserID, id)
		if err != nil {
			return nil, err
		}
	default:
		root, err = s.nodeRepo.GetRootByOwner(ctx, actor.UserID)
		if err != nil {
			return nil, err
		}
	}

	if err := s.policy.allowRead(actor, root); err != nil {
		return nil, err
	}

	maxDepth := req.MaxDepth
	if maxDepth <= 0 {
		maxDepth = defaultMaxDepth
	}
	includeDeleted := req.IncludeDeleted && actor.IsOwner()

	nodes, err := s.nodeRepo.ListSubtree(ctx, actor.UserID, root, maxDepth, includeDeleted)
	if err != nil {
		return nil, err
	}

	// Batch-load blobs for file nodes so the tree response includes size/mime.
	var fileIDs []valueobject.NodeID
	for _, n := range nodes {
		if n.IsFile() {
			fileIDs = append(fileIDs, n.ID())
		}
	}
	blobMap, err := s.blobRepo.GetByNodeIDs(ctx, fileIDs)
	if err != nil {
		return nil, fmt.Errorf("load file blobs for tree: %w", err)
	}

	return buildTree(root, nodes, blobMap), nil
}

// buildTree assembles a tree from a flat list (root first, then descendants).
func buildTree(root *entity.Node, nodes []*entity.Node, blobMap map[valueobject.NodeID]*entity.FileBlob) *dto.TreeNodeResponse {
	byParent := map[string][]*entity.Node{}
	rootKey := root.ID().String()
	var rootNode *entity.Node
	for _, n := range nodes {
		if n.ID().Equals(root.ID()) {
			rootNode = n
			continue
		}
		if pid := n.ParentID(); pid != nil {
			pidS := pid.String()
			byParent[pidS] = append(byParent[pidS], n)
		}
	}
	if rootNode == nil {
		rootNode = root
	}

	var build func(n *entity.Node) dto.TreeNodeResponse
	build = func(n *entity.Node) dto.TreeNodeResponse {
		resp := dto.TreeNodeResponse{NodeResponse: *toNodeResponse(n, blobMap[n.ID()])}
		for _, child := range byParent[n.ID().String()] {
			resp.Children = append(resp.Children, build(child))
		}
		return resp
	}
	_ = rootKey
	return ptrTree(build(rootNode))
}

func ptrTree(t dto.TreeNodeResponse) *dto.TreeNodeResponse { return &t }
