// Package ctxkit 在 context 里存取请求级信息（请求 ID、当前用户等），
// 避免各层用裸 string key、也避免循环依赖。
package ctxkit

import "context"

type ctxKey int

const (
	requestIDKey ctxKey = iota
	userIDKey
)

func WithRequestID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, requestIDKey, id)
}

func RequestID(ctx context.Context) string {
	if v, ok := ctx.Value(requestIDKey).(string); ok {
		return v
	}
	return ""
}

func WithUserID(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func UserID(ctx context.Context) string {
	if v, ok := ctx.Value(userIDKey).(string); ok {
		return v
	}
	return ""
}
