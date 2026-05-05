package handler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/dto"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/usecase"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
)

// ActorExtractor decouples the handler from a specific middleware package.
type ActorExtractor func(*http.Request) *port.Actor

// AIHandler exposes the four /ai/v1 HTTP endpoints.
type AIHandler struct {
	uc       port.AIUseCase
	getActor ActorExtractor
}

func NewAIHandler(uc port.AIUseCase, getActor ActorExtractor) *AIHandler {
	return &AIHandler{uc: uc, getActor: getActor}
}

func (h *AIHandler) actor(r *http.Request) *port.Actor {
	return h.getActor(r)
}

// PlanCommand handles POST /ai/v1/commands.
//
// Body: {"input": "..."}.
// Response (201): {status:success,data:{<CommandResponse with status=awaiting_confirmation>}}.
func (h *AIHandler) PlanCommand(w http.ResponseWriter, r *http.Request) {
	var req dto.PlanCommandRequest
	if err := decodeJSON(w, r, &req); err != nil {
		http.Error(w, "invalid request body", http.StatusBadRequest)
		return
	}
	if err := req.Validate(); err != nil {
		SendError(w, err)
		return
	}

	actor := h.actor(r)
	cmd, err := h.uc.Plan(r.Context(), actor, req.Input)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, usecase.ToCommandResponse(cmd), http.StatusCreated)
}

// ExecuteCommand handles POST /ai/v1/commands/{id}/execute.
//
// Body: empty for now.
// Response (200): {status:success,data:{<CommandResponse with status=executed|failed>}}.
func (h *AIHandler) ExecuteCommand(w http.ResponseWriter, r *http.Request) {
	id, err := commandIDFromURL(r)
	if err != nil {
		SendError(w, err)
		return
	}
	actor := h.actor(r)
	cmd, err := h.uc.Execute(r.Context(), actor, id)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, usecase.ToCommandResponse(cmd), http.StatusOK)
}

// CancelCommand handles POST /ai/v1/commands/{id}/cancel.
func (h *AIHandler) CancelCommand(w http.ResponseWriter, r *http.Request) {
	id, err := commandIDFromURL(r)
	if err != nil {
		SendError(w, err)
		return
	}
	actor := h.actor(r)
	cmd, err := h.uc.Cancel(r.Context(), actor, id)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, usecase.ToCommandResponse(cmd), http.StatusOK)
}

// GetCommand handles GET /ai/v1/commands/{id}.
func (h *AIHandler) GetCommand(w http.ResponseWriter, r *http.Request) {
	id, err := commandIDFromURL(r)
	if err != nil {
		SendError(w, err)
		return
	}
	actor := h.actor(r)
	cmd, err := h.uc.Get(r.Context(), actor, id)
	if err != nil {
		SendError(w, err)
		return
	}
	SendSuccess(w, usecase.ToCommandResponse(cmd), http.StatusOK)
}

func commandIDFromURL(r *http.Request) (valueobject.CommandID, error) {
	raw := chi.URLParam(r, "id")
	if raw == "" {
		return valueobject.CommandID{}, dto.ErrCommandIDRequired
	}
	id, err := valueobject.ParseCommandID(raw)
	if err != nil {
		// Surface as a 400 INVALID_COMMAND_ID consistently.
		return valueobject.CommandID{}, domainerr.New(
			domainerr.CodeInvalidCommandID,
			"invalid command_id",
			err,
		)
	}
	return id, nil
}

func decodeJSON(w http.ResponseWriter, r *http.Request, dst any) error {
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		return err
	}
	return nil
}

