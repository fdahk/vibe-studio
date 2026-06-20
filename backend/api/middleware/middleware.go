// Package middleware 全局 HTTP 中间件，采用 net/http 经典模式 func(http.Handler) http.Handler。
package middleware

import (
	"log/slog"
	"net/http"
	"runtime/debug"
	"time"

	"github.com/google/uuid"

	"vibe-studio/backend/pkg/ctxkit"
	"vibe-studio/backend/pkg/errorx"
	"vibe-studio/backend/pkg/response"
)

const RequestIDHeader = "X-Request-ID"

// Middleware 经典中间件类型：包裹一个 handler 返回新 handler。
type Middleware func(http.Handler) http.Handler

// Chain 按顺序套用中间件（第一个在最外层）。
func Chain(h http.Handler, mws ...Middleware) http.Handler {
	for i := len(mws) - 1; i >= 0; i-- {
		h = mws[i](h)
	}
	return h
}

// statusRecorder 记录响应状态码，供访问日志使用。
type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(code int) {
	r.status = code
	r.ResponseWriter.WriteHeader(code)
}

// Recovery 捕获 panic，返回统一错误响应（最外层）。
func Recovery() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					slog.Error("panic recovered", "err", rec, "stack", string(debug.Stack()))
					response.Fail(w, errorx.ErrInternal)
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

// RequestID 生成/透传 X-Request-ID，并写入 context。
func RequestID() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			rid := r.Header.Get(RequestIDHeader)
			if rid == "" {
				rid = uuid.NewString()
			}
			w.Header().Set(RequestIDHeader, rid)
			next.ServeHTTP(w, r.WithContext(ctxkit.WithRequestID(r.Context(), rid)))
		})
	}
}

// CORS 跨域中间件：只回显白名单内的 Origin 并允许携带 cookie 凭证。
// 带 credentials 时浏览器禁止 `*`，故必须回显具体 Origin。
func CORS(allowedOrigins []string) Middleware {
	allowed := make(map[string]bool, len(allowedOrigins))
	for _, o := range allowedOrigins {
		allowed[o] = true
	}
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if origin := r.Header.Get("Origin"); origin != "" && allowed[origin] {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Add("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, "+RequestIDHeader)
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// AccessLog 打印结构化访问日志（方法/路径/状态码/耗时/请求ID）。
func AccessLog() Middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			slog.Info("access",
				"method", r.Method, "path", r.URL.Path, "status", rec.status,
				"dur", time.Since(start).String(), "rid", ctxkit.RequestID(r.Context()))
		})
	}
}
