package handler

import (
	"net/http"

	"vibe-studio/backend/infra"
	"vibe-studio/backend/pkg/response"
)

// HealthHandler 提供存活/就绪探针（K8s 风格）。
type HealthHandler struct {
	deps *infra.Deps
}

func NewHealthHandler(deps *infra.Deps) *HealthHandler {
	return &HealthHandler{deps: deps}
}

// Healthz 存活探针：进程活着就返回 200，不依赖任何外部服务。
func (h *HealthHandler) Healthz(w http.ResponseWriter, r *http.Request) {
	response.OK(w, map[string]string{"status": "ok"})
}

// Readyz 就绪探针：检查 mysql/redis/minio，任一不可用返回 503。
func (h *HealthHandler) Readyz(w http.ResponseWriter, r *http.Request) {
	checks := h.deps.HealthCheck(r.Context())
	ready := true
	for _, ok := range checks {
		if !ok {
			ready = false
		}
	}
	status := http.StatusOK
	if !ready {
		status = http.StatusServiceUnavailable
	}
	response.WriteJSON(w, status, map[string]any{"ready": ready, "checks": checks})
}
