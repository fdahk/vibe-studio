package user

import (
	"net/http"

	"vibe-studio/backend/api/openapi"
	userapp "vibe-studio/backend/application/user"
	domain "vibe-studio/backend/domain/user"
	"vibe-studio/backend/pkg/errorx"
	"vibe-studio/backend/pkg/response"
)

// refreshCookiePath：refresh cookie 只在鉴权端点携带，不污染其它请求。
const refreshCookiePath = "/api/v1/auth"

// CookieConfig refresh cookie 的属性（来自 conf.Session）。
type CookieConfig struct {
	Name   string
	Secure bool
	Domain string
	MaxAge int // 秒
}

// 会话相关依赖由组合根注入。
var (
	sessionSvc *userapp.SessionService
	cookieCfg  CookieConfig
)

func SetSession(svc *userapp.SessionService, cookie CookieConfig) {
	sessionSvc = svc
	cookieCfg = cookie
}

func setRefreshCookie(w http.ResponseWriter, token string) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieCfg.Name,
		Value:    token,
		Path:     refreshCookiePath,
		Domain:   cookieCfg.Domain,
		MaxAge:   cookieCfg.MaxAge,
		HttpOnly: true,
		Secure:   cookieCfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func clearRefreshCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     cookieCfg.Name,
		Value:    "",
		Path:     refreshCookiePath,
		Domain:   cookieCfg.Domain,
		MaxAge:   -1,
		HttpOnly: true,
		Secure:   cookieCfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func readRefreshCookie(r *http.Request) string {
	c, err := r.Cookie(cookieCfg.Name)
	if err != nil {
		return ""
	}
	return c.Value
}

// issueSession 登录成功后建会话：种 refresh cookie + 响应体返回 access。
func issueSession(w http.ResponseWriter, r *http.Request, u *domain.User) {
	if sessionSvc == nil {
		response.Fail(w, errorx.ErrInternal)
		return
	}
	pair, err := sessionSvc.Issue(r.Context(), u.ID, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		response.Fail(w, errorx.ErrInternal.Wrap(err))
		return
	}
	setRefreshCookie(w, pair.Refresh)
	response.OK(w, openapi.AuthData{User: toModel(u), AccessToken: pair.Access})
}
