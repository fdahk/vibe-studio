// Package router 是组合根（composition root）：注册全局探针、装配共享能力与各业务域的依赖，
// 再聚合调用各域自己声明的路由。这里**不含任何单个域的路由细节**——那些归各域模块自己。
package router

import (
	"log/slog"
	"net/http"
	"time"

	"vibe-studio/backend/api/handler"
	useruser "vibe-studio/backend/api/handler/user"
	"vibe-studio/backend/api/middleware"
	userapp "vibe-studio/backend/application/user"
	"vibe-studio/backend/infra"
	dbmigrate "vibe-studio/backend/infra/migrate"
	"vibe-studio/backend/infra/oauth"
	"vibe-studio/backend/infra/persistence"
	"vibe-studio/backend/infra/sms"
	"vibe-studio/backend/pkg/auth"
)

// Register 装配并聚合路由。
func Register(mux *http.ServeMux, deps *infra.Deps) {
	// 全局健康探针（基础设施关注点，不属于任何业务域）。
	health := handler.NewHealthHandler(deps)
	mux.HandleFunc("GET /healthz", health.Healthz)
	mux.HandleFunc("GET /readyz", health.Readyz)
	mux.HandleFunc("GET /api/v1/health", health.Healthz)

	if deps.Cfg == nil {
		slog.Warn("配置不可用，业务域未装配")
		return
	}

	// 共享鉴权能力：access JWT（短时），供 Auth 中间件(校验) 与会话服务(签发) 复用。
	jwt := auth.NewJWT(deps.Cfg.JWT.Secret, time.Duration(deps.Cfg.Session.AccessTTLMinutes)*time.Minute)
	middleware.SetJWT(jwt)

	// ===== 各业务域装配：组合根注入依赖，路由由各域自己声明 =====
	registerUserDomain(mux, deps, jwt)
	// 未来新增领域：在此加一行 registerXxxDomain(...)，路由细节写在对应模块里。
}

// registerUserDomain 装配 user 域的依赖（含 DB 迁移），并调用其自有路由声明。
func registerUserDomain(mux *http.ServeMux, deps *infra.Deps, jwt *auth.JWT) {
	if deps.DB == nil {
		slog.Warn("DB 不可用，user 域未装配(degraded)")
		return
	}
	// 跑数据库迁移（golang-migrate，版本化 SQL）。
	if sqlDB, err := deps.DB.DB(); err != nil {
		slog.Error("获取 sql.DB 失败", "err", err)
	} else if err := dbmigrate.Run(sqlDB); err != nil {
		slog.Error("数据库迁移失败", "err", err)
	}

	// 短信：dev 用 console 发送器（验证码打日志）；验证码存 Redis（TTL/冷却来自配置）。
	repo := persistence.NewRepo(deps.DB)
	sender := sms.NewConsoleSender()
	codeStore := sms.NewRedisCodeStore(
		deps.Redis,
		time.Duration(deps.Cfg.SMS.CodeTTLSeconds)*time.Second,
		time.Duration(deps.Cfg.SMS.CooldownSeconds)*time.Second,
	)
	useruser.SetService(userapp.NewService(repo, sender, codeStore))

	// 会话：refresh 存 DB sessions 表（轮换 + 复用检测），access 由 jwt 签发。
	sessionRepo := persistence.NewSessionRepo(deps.DB)
	sessionSvc := userapp.NewSessionService(sessionRepo, jwt, time.Duration(deps.Cfg.Session.RefreshTTLDays)*24*time.Hour)
	useruser.SetSession(sessionSvc, useruser.CookieConfig{
		Name:   deps.Cfg.Session.CookieName,
		Secure: deps.Cfg.Session.CookieSecure,
		Domain: deps.Cfg.Session.CookieDomain,
		MaxAge: deps.Cfg.Session.RefreshTTLDays * 24 * 60 * 60,
	})

	// 第三方登录：GitHub OAuth（未配 client 则优雅关闭）；state 走 Redis 防 CSRF。
	gh := oauth.NewGitHub(deps.Cfg.OAuth.GitHubClientID, deps.Cfg.OAuth.GitHubClientSecret, deps.Cfg.OAuth.GitHubRedirectURL)
	stateStore := oauth.NewStateStore(deps.Redis, 10*time.Minute)
	useruser.SetOAuth(gh, stateStore, deps.Cfg.OAuth.FrontendURL)

	useruser.Routes(mux) // ← user 路由归属 user 模块，这里只聚合
	slog.Info("user 域已装配")
}
