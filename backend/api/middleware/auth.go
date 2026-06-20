package middleware

import (
	"net/http"
	"strings"

	"vibe-studio/backend/pkg/auth"
	"vibe-studio/backend/pkg/ctxkit"
	"vibe-studio/backend/pkg/errorx"
	"vibe-studio/backend/pkg/response"
)

var jwtParser *auth.JWT

// SetJWT 注入 JWT 校验器（组合根调用）。
func SetJWT(j *auth.JWT) { jwtParser = j }

// Auth 鉴权中间件：解析 Authorization: Bearer <token> → 校验 → userID 写入 context。
func Auth() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authz := r.Header.Get("Authorization")
			token := strings.TrimPrefix(authz, "Bearer ")
			if token == "" || token == authz { // 空 或 没有 Bearer 前缀
				response.Fail(w, errorx.ErrUnauthorized)
				return
			}
			uid, err := jwtParser.Parse(token)
			if err != nil {
				response.Fail(w, err)
				return
			}
			next.ServeHTTP(w, r.WithContext(ctxkit.WithUserID(r.Context(), uid)))
		})
	}
}
