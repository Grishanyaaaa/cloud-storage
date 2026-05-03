package usecase

import (
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/entity"
)

// authorizationPolicy concentrates per-action authorization decisions.
//
// Rules:
//   - Owner: all actions on owned nodes are allowed.
//   - ShareLink: only operations covered by Permission are allowed AND only
//     within the share-root's subtree.
type authorizationPolicy struct{}

func newAuthorizationPolicy() *authorizationPolicy { return &authorizationPolicy{} }

// allowOwner enforces the action is performed by the resource owner.
// All hierarchy-mutating actions (create folder, move, generate upload URL,
// finalize, abort, restore, share-management) MUST require the owner.
func (p *authorizationPolicy) allowOwner(actor *port.Actor, target *entity.Node) error {
	if actor == nil || !actor.IsOwner() {
		return domainerr.ErrForbidden
	}
	if target != nil && !target.OwnerID().Equals(actor.UserID) {
		return domainerr.ErrForbidden
	}
	return nil
}

// allowRead enforces a read action.
// - Owner of target: ok.
// - ShareLink with view|edit permission: ok if target is inside share scope.
func (p *authorizationPolicy) allowRead(actor *port.Actor, target *entity.Node) error {
	if actor == nil || target == nil {
		return domainerr.ErrForbidden
	}
	if actor.IsOwner() {
		if !target.OwnerID().Equals(actor.UserID) {
			return domainerr.ErrForbidden
		}
		return nil
	}
	if !actor.IsShareLink() {
		return domainerr.ErrForbidden
	}
	if actor.Share == nil || actor.ShareRoot == nil {
		return domainerr.ErrForbidden
	}
	if !actor.Share.Permission().AllowsRead() {
		return domainerr.ErrPermissionDenied
	}
	return actor.Share.AssertCovers(actor.ShareRoot, target)
}

// allowRename enforces rename-permission semantics.
func (p *authorizationPolicy) allowRename(actor *port.Actor, target *entity.Node) error {
	if actor == nil || target == nil {
		return domainerr.ErrForbidden
	}
	if actor.IsOwner() {
		if !target.OwnerID().Equals(actor.UserID) {
			return domainerr.ErrForbidden
		}
		return nil
	}
	if !actor.IsShareLink() {
		return domainerr.ErrForbidden
	}
	if actor.Share == nil || actor.ShareRoot == nil {
		return domainerr.ErrForbidden
	}
	if !actor.Share.Permission().AllowsRename() {
		return domainerr.ErrPermissionDenied
	}
	return actor.Share.AssertCovers(actor.ShareRoot, target)
}

// allowDelete enforces delete-permission semantics.
func (p *authorizationPolicy) allowDelete(actor *port.Actor, target *entity.Node) error {
	if actor == nil || target == nil {
		return domainerr.ErrForbidden
	}
	if actor.IsOwner() {
		if !target.OwnerID().Equals(actor.UserID) {
			return domainerr.ErrForbidden
		}
		return nil
	}
	if !actor.IsShareLink() {
		return domainerr.ErrForbidden
	}
	if actor.Share == nil || actor.ShareRoot == nil {
		return domainerr.ErrForbidden
	}
	if !actor.Share.Permission().AllowsDelete() {
		return domainerr.ErrPermissionDenied
	}
	return actor.Share.AssertCovers(actor.ShareRoot, target)
}
