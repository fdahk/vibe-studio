// Package router 是组合根（composition root）：注册全局探针、装配共享能力与各业务域的依赖，
// 再聚合调用各业务域自己声明的路由。这里**不含任何单个域的路由细节**——那些归各域模块自己。
package router

import (
	"net/http"

	"vibe-studio/backend/api/handler"
	"vibe-studio/backend/infra"
)

// Register 装配并聚合路由。
func Register(mux *http.ServeMux, deps *infra.Deps) {
	// 全局健康探针（基础设施关注点，不属于任何业务域）。
	health := handler.NewHealthHandler(deps)
	mux.HandleFunc("GET /healthz", health.Healthz)
	mux.HandleFunc("GET /readyz", health.Readyz)
	mux.HandleFunc("GET /api/v1/health", health.Healthz)

	// ===== 各业务域在此装配（组合根注入依赖，路由由各域自己声明）=====
	// 例：registerUserDomain(mux, deps, jwt) —— 待对应业务域完成后接入。
}
