package port

import "github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"

// ActorKind enumerates kinds of principals.
// In ai-service we currently only accept JWT-authenticated owners — there is
// no share-link concept (those operations are tied to a single user's tree).
type ActorKind string

const (
	ActorKindOwner ActorKind = "owner"
)

// Actor is the principal performing an ai-service action.
//
//	UserID — owner's id (extracted from JWT claims)
//	Roles  — claims-derived roles (currently unused; reserved for admin-only ops)
//	JWT    — raw access token to be propagated to storage-service as Authorization header
type Actor struct {
	Kind   ActorKind
	UserID valueobject.UserID
	Roles  []string
	JWT    string
}

// IsOwner returns true when the actor is the resource owner.
func (a *Actor) IsOwner() bool { return a != nil && a.Kind == ActorKindOwner }
