package handler

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/storage-service/internal/domain/domainerr"
)

const maxJSONBody = 1 << 20 // 1 MB

// ActorExtractor decouples the handler from a specific middleware package.
type ActorExtractor func(*http.Request) *port.Actor

// StorageHandler exposes the storage HTTP API.
type StorageHandler struct {
	useCase     port.StorageUseCase
	getActor    ActorExtractor
}

func NewStorageHandler(useCase port.StorageUseCase, getActor ActorExtractor) *StorageHandler {
	return &StorageHandler{useCase: useCase, getActor: getActor}
}

// ----- helpers ---------------------------------------------------------------

func (h *StorageHandler) actor(r *http.Request) *port.Actor {
	return h.getActor(r)
}

func decodeJSON(w http.ResponseWriter, r *http.Request, v any) error {
	r.Body = http.MaxBytesReader(w, r.Body, maxJSONBody)
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		return err
	}
	return nil
}

func boolParam(r *http.Request, key string) bool {
	v := r.URL.Query().Get(key)
	if v == "" {
		return false
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return false
	}
	return b
}

func intParam(r *http.Request, key string) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return 0
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return 0
	}
	return n
}

// ----- owner endpoints -------------------------------------------------------

// EnsureRoot ensures (and lazily creates) the user's root folder.
// POST /storage/v1/me/root
func (h *StorageHandler) EnsureRoot(w http.ResponseWriter, r *http.Request) {
	resp, err := h.useCase.EnsureUserRoot(r.Context(), h.actor(r))
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusOK)
}

// GetTree returns a partial tree.
// GET /storage/v1/tree?root_id=&max_depth=&include_deleted=
func (h *StorageHandler) GetTree(w http.ResponseWriter, r *http.Request) {
	req := dto.GetTreeRequest{
		RootID:         r.URL.Query().Get("root_id"),
		MaxDepth:       intParam(r, "max_depth"),
		IncludeDeleted: boolParam(r, "include_deleted"),
	}
	resp, err := h.useCase.GetTree(r.Context(), h.actor(r), req)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusOK)
}

// GetNode returns a single node.
// GET /storage/v1/nodes/{id}
func (h *StorageHandler) GetNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	resp, err := h.useCase.GetNode(r.Context(), h.actor(r), id)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusOK)
}

// ListChildren paginates children of a folder.
// GET /storage/v1/folders/{id}/children
func (h *StorageHandler) ListChildren(w http.ResponseWriter, r *http.Request) {
	req := dto.ListChildrenRequest{
		ParentID:       chi.URLParam(r, "id"),
		Cursor:         r.URL.Query().Get("cursor"),
		Limit:          intParam(r, "limit"),
		IncludeDeleted: boolParam(r, "include_deleted"),
	}
	resp, err := h.useCase.ListChildren(r.Context(), h.actor(r), req)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusOK)
}

// CreateFolder creates a folder.
// POST /storage/v1/folders
func (h *StorageHandler) CreateFolder(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateFolderRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.useCase.CreateFolder(r.Context(), h.actor(r), req)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusCreated)
}

// RenameNode renames a node.
// PATCH /storage/v1/nodes/{id}/rename
func (h *StorageHandler) RenameNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.RenameNodeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.useCase.RenameNode(r.Context(), h.actor(r), id, req)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusOK)
}

// MoveNode moves a node to a new parent.
// PATCH /storage/v1/nodes/{id}/move
func (h *StorageHandler) MoveNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var req dto.MoveNodeRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.useCase.MoveNode(r.Context(), h.actor(r), id, req)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusOK)
}

// SoftDeleteNode soft-deletes a node and its subtree.
// DELETE /storage/v1/nodes/{id}
func (h *StorageHandler) SoftDeleteNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := h.useCase.SoftDeleteNode(r.Context(), h.actor(r), id); err != nil {
		SendError(w, err)
		return
	}
	SendNoContent(w)
}

// RestoreNode restores a node and its subtree.
// POST /storage/v1/nodes/{id}/restore
func (h *StorageHandler) RestoreNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	resp, err := h.useCase.RestoreNode(r.Context(), h.actor(r), id)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusOK)
}

// GenerateUploadURL produces a pre-signed PUT URL for a brand-new file.
// POST /storage/v1/files/upload-url
func (h *StorageHandler) GenerateUploadURL(w http.ResponseWriter, r *http.Request) {
	var req dto.GenerateUploadURLRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	resp, err := h.useCase.GenerateUploadURL(r.Context(), h.actor(r), req)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusOK)
}

// FinalizeUpload activates a previously generated pre-signed PUT.
// POST /storage/v1/files/{id}/finalize
func (h *StorageHandler) FinalizeUpload(w http.ResponseWriter, r *http.Request) {
	var req dto.FinalizeUploadRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.NodeID = chi.URLParam(r, "id")
	resp, err := h.useCase.FinalizeUpload(r.Context(), h.actor(r), req)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusOK)
}

// AbortUpload aborts a pending upload.
// POST /storage/v1/files/{id}/abort
func (h *StorageHandler) AbortUpload(w http.ResponseWriter, r *http.Request) {
	req := dto.AbortUploadRequest{NodeID: chi.URLParam(r, "id")}
	if err := h.useCase.AbortUpload(r.Context(), h.actor(r), req); err != nil {
		SendError(w, err)
		return
	}
	SendNoContent(w)
}

// GenerateDownloadURL returns a pre-signed GET URL.
// GET /storage/v1/files/{id}/download-url?disposition=
func (h *StorageHandler) GenerateDownloadURL(w http.ResponseWriter, r *http.Request) {
	req := dto.GenerateDownloadURLRequest{
		NodeID:      chi.URLParam(r, "id"),
		Disposition: r.URL.Query().Get("disposition"),
	}
	resp, err := h.useCase.GenerateDownloadURL(r.Context(), h.actor(r), req)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusOK)
}

// CreateShareLink creates a share-link.
// POST /storage/v1/nodes/{id}/shares
func (h *StorageHandler) CreateShareLink(w http.ResponseWriter, r *http.Request) {
	var req dto.CreateShareRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	req.NodeID = chi.URLParam(r, "id")
	resp, err := h.useCase.CreateShareLink(r.Context(), h.actor(r), req)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusCreated)
}

// ListShareLinks lists shares attached to a node.
// GET /storage/v1/nodes/{id}/shares?include_revoked=
func (h *StorageHandler) ListShareLinks(w http.ResponseWriter, r *http.Request) {
	req := dto.ListSharesRequest{
		NodeID:         chi.URLParam(r, "id"),
		IncludeRevoked: boolParam(r, "include_revoked"),
	}
	resp, err := h.useCase.ListShareLinks(r.Context(), h.actor(r), req)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, resp, http.StatusOK)
}

// RevokeShareLink revokes a share by ID.
// DELETE /storage/v1/shares/{id}
func (h *StorageHandler) RevokeShareLink(w http.ResponseWriter, r *http.Request) {
	req := dto.RevokeShareRequest{ShareID: chi.URLParam(r, "id")}
	if err := h.useCase.RevokeShareLink(r.Context(), h.actor(r), req); err != nil {
		SendError(w, err)
		return
	}
	SendNoContent(w)
}

// ----- public (share-token) endpoints ----------------------------------------

// PublicShareInfo returns metadata about the share itself.
// GET /storage/v1/public/{token}
func (h *StorageHandler) PublicShareInfo(w http.ResponseWriter, r *http.Request) {
	actor := h.actor(r)
	if actor == nil || !actor.IsShareLink() {
		SendError(w, domainerr.ErrForbidden)
		return
	}
	resp := dto.PublicShareResponse{
		NodeID:     actor.ShareRoot.ID().String(),
		Kind:       string(actor.ShareRoot.Kind()),
		Name:       actor.ShareRoot.Name().String(),
		Permission: actor.Share.Permission().String(),
		ExpiresAt:  actor.Share.ExpiresAt(),
	}
	SendSuccess(w, resp, http.StatusOK)
}
