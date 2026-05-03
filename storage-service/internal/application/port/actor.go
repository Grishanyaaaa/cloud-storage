package port

import (
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/valueobject"
)

// ActorKind enumerates kinds of principals.
// Owner — JWT-authenticated owner of the resource.
// ShareLink — anonymous client presenting a valid share-token.
type ActorKind string

const (
	ActorKindOwner     ActorKind = "owner"
	ActorKindShareLink ActorKind = "share_link"
)

// Actor is the principal performing a storage action.
// For ActorKindShareLink:
//   - UserID is the owner of the share (the resource's owner) — every read inside
//     the use case is performed against this user's tree;
//   - Share is the resolved entity (already verified as active);
//   - ShareRoot is the entity at Share.NodeID — used to bound subtree access.
type Actor struct {
	Kind      ActorKind
	UserID    valueobject.UserID
	Roles     []string
	Share     *entity.Share // nil for ActorKindOwner
	ShareRoot *entity.Node  // nil for ActorKindOwner
}

// IsOwner returns true when actor is the resource owner.
func (a *Actor) IsOwner() bool { return a != nil && a.Kind == ActorKindOwner }

// IsShareLink returns true when actor is a share-link consumer.
func (a *Actor) IsShareLink() bool { return a != nil && a.Kind == ActorKindShareLink }
