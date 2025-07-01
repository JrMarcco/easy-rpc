package easyrpc

import "context"

type contextKeyOneway struct{}

func ContextWithOneway(ctx context.Context) context.Context {
	return context.WithValue(ctx, contextKeyOneway{}, true)
}

func isOneway(ctx context.Context) bool {
	val, ok := ctx.Value(contextKeyOneway{}).(bool)
	return ok && val
}
