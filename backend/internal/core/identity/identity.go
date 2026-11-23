// Package identity carries the authenticated actor through request contexts.
// Values are set only by auth middleware.
package identity

import "context"

// ActorType discriminates the auth context a session belongs to.
type ActorType string

// Known actor types.
const (
	ActorPlatformUser ActorType = "platform_user"
	ActorUser         ActorType = "user"
	ActorTerminal     ActorType = "terminal"
	ActorSystem       ActorType = "system"
)

// Actor is the authenticated principal of the current request.
type Actor struct {
	Type      ActorType
	ID        uint64 // platform_users.id or users.id
	SessionID uint64
	TenantID  uint64 // 0 for platform actors
	Role      string // platform role, or tenant role code
	RoleID    uint64 // tenant roles.id (0 for platform actors)
	Access    Access // tenant users only
}

// IsZero reports whether there is no authenticated actor.
func (a Actor) IsZero() bool { return a.Type == "" }

type ctxKey struct{}

// WithActor stores the authenticated actor in ctx.
func WithActor(ctx context.Context, a Actor) context.Context {
	return context.WithValue(ctx, ctxKey{}, a)
}

// ActorFrom returns the actor stored by WithActor.
func ActorFrom(ctx context.Context) (Actor, bool) {
	v, ok := ctx.Value(ctxKey{}).(Actor)
	return v, ok
}
