package tenancy

import "context"

type Context struct {
	TenantID string
	ActorID  string
	Roles    []string
	Scopes   []string
}

type key struct{}

func WithContext(ctx context.Context, tc Context) context.Context {
	return context.WithValue(ctx, key{}, tc)
}

func FromContext(ctx context.Context) (Context, bool) {
	tc, ok := ctx.Value(key{}).(Context)
	return tc, ok
}

func Require(ctx context.Context) (Context, error) {
	tc, ok := FromContext(ctx)
	if !ok || tc.TenantID == "" {
		return Context{}, ErrMissingTenant
	}
	return tc, nil
}

type tenantError string

func (e tenantError) Error() string { return string(e) }

const ErrMissingTenant tenantError = "tenant context is required"
