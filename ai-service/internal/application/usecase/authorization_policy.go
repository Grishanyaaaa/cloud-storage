package usecase

import (
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/entity"
)

// authorizationPolicy concentrates per-action authorization decisions for ai-service.
//
// Rules (MVP):
//   - Only the owner of the ai_command may Plan/Execute/Cancel/Get it.
//   - Public share-links are not authorized to use ai-service at all.
type authorizationPolicy struct{}

func newAuthorizationPolicy() *authorizationPolicy { return &authorizationPolicy{} }

// allowOwner enforces the action is performed by an owner-actor.
// If `target` is non-nil, also enforces target.UserID == actor.UserID.
func (p *authorizationPolicy) allowOwner(actor *port.Actor, target *entity.AiCommand) error {
	if actor == nil || !actor.IsOwner() {
		return domainerr.ErrForbidden
	}
	if target == nil {
		return nil
	}
	return target.AssertOwner(actor.UserID)
}
