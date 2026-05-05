package middleware

import (
	"context"
	"net/http"

	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/application/port"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/domainerr"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/domain/valueobject"
	"github.com/Grishanyaaaa/cloud-storage/ai-service/internal/presentation/http/handler"
)

type ctxKey struct{ name string }

var actorCtxKey = ctxKey{name: "actor"}

// WithActor stores an actor in context.
func WithActor(ctx context.Context, actor *port.Actor) context.Context {
	return context.WithValue(ctx, actorCtxKey, actor)
}

// ActorFromContext retrieves an actor previously stored by middleware.
// Returns nil when missing.
func ActorFromContext(ctx context.Context) *port.Actor {
	v := ctx.Value(actorCtxKey)
	if v == nil {
		return nil
	}
	if a, ok := v.(*port.Actor); ok {
		return a
	}
	return nil
}

// AuthContext validates the incoming JWT and injects an owner Actor into the request context.
//
// Unlike storage-service we ALSO preserve the raw access token in actor.JWT so
// that the use-case layer can propagate it to storage-service. (ai-service
// has no service-to-service credentials of its own.)
func AuthContext(parser port.JWTParser) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			payload, err := parser.Parse(r)
			if err != nil {
				handler.SendError(w, err)
				return
			}
			uid, err := valueobject.ParseUserID(payload.UserID)
			if err != nil {
				handler.SendError(w, domainerr.ErrInvalidToken)
				return
			}
			actor := &port.Actor{
				Kind:   port.ActorKindOwner,
				UserID: uid,
				Roles:  append([]string(nil), payload.Roles...),
				JWT:    payload.RawToken,
			}
			next.ServeHTTP(w, r.WithContext(WithActor(r.Context(), actor)))
		})
	}
}
