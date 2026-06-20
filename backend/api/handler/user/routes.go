package user

import (
	"net/http"

	"vibe-studio/backend/api/middleware"
)

// Routes 声明 user 域自己的路由（路由归属本模块，集中在此；服务由 SetService 注入）。
// 组合根只负责"调用各模块的 Routes 做聚合"，不掺入任何域的路由细节。
func Routes(mux *http.ServeMux) {
	// 公开
	mux.HandleFunc("POST /api/v1/auth/register", Register)
	mux.HandleFunc("POST /api/v1/auth/login", Login)
	mux.HandleFunc("POST /api/v1/auth/sms/code", SendSMSCode)
	mux.HandleFunc("POST /api/v1/auth/login/phone", PhoneLogin)
	mux.HandleFunc("POST /api/v1/auth/refresh", Refresh)
	mux.HandleFunc("POST /api/v1/auth/logout", Logout)
	// GitHub OAuth：浏览器跳转流程（非 JSON API）
	mux.HandleFunc("GET /api/v1/auth/oauth/github", OAuthGitHubLogin)
	mux.HandleFunc("GET /api/v1/auth/oauth/github/callback", OAuthGitHubCallback)
	// 受保护：仅此路由套 Auth 中间件
	mux.Handle("GET /api/v1/users/me", middleware.Auth()(http.HandlerFunc(GetMe)))
}
