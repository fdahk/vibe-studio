// user 域的 HTTP handler（net/http）。请求/响应类型来自 OpenAPI 生成的 openapi 包，
// 认证委托给 application 的 Service，会话签发委托给 SessionService。
package user

import (
	"encoding/json"
	"net/http"

	"vibe-studio/backend/api/openapi"
	"vibe-studio/backend/pkg/ctxkit"
	"vibe-studio/backend/pkg/errorx"
	"vibe-studio/backend/pkg/response"
)

// Register POST /api/v1/auth/register
func Register(w http.ResponseWriter, r *http.Request) {
	var req openapi.RegisterRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, errorx.ErrBadRequest.Wrap(err))
		return
	}
	if svc == nil {
		response.Fail(w, errorx.ErrInternal)
		return
	}
	email := ""
	if req.Email != nil {
		email = *req.Email
	}
	u, err := svc.Register(r.Context(), req.Username, req.Password, email)
	if err != nil {
		response.Fail(w, err)
		return
	}
	issueSession(w, r, u)
}

// Login POST /api/v1/auth/login
func Login(w http.ResponseWriter, r *http.Request) {
	var req openapi.LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, errorx.ErrBadRequest.Wrap(err))
		return
	}
	if svc == nil {
		response.Fail(w, errorx.ErrInternal)
		return
	}
	u, err := svc.Login(r.Context(), req.Username, req.Password)
	if err != nil {
		response.Fail(w, err)
		return
	}
	issueSession(w, r, u)
}

// SendSMSCode POST /api/v1/auth/sms/code
func SendSMSCode(w http.ResponseWriter, r *http.Request) {
	var req openapi.SmsCodeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, errorx.ErrBadRequest.Wrap(err))
		return
	}
	if svc == nil {
		response.Fail(w, errorx.ErrInternal)
		return
	}
	if err := svc.SendSMSCode(r.Context(), req.Phone); err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, nil)
}

// PhoneLogin POST /api/v1/auth/login/phone （新手机号自动注册）
func PhoneLogin(w http.ResponseWriter, r *http.Request) {
	var req openapi.PhoneLoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Fail(w, errorx.ErrBadRequest.Wrap(err))
		return
	}
	if svc == nil {
		response.Fail(w, errorx.ErrInternal)
		return
	}
	u, err := svc.LoginByPhone(r.Context(), req.Phone, req.Code)
	if err != nil {
		response.Fail(w, err)
		return
	}
	issueSession(w, r, u)
}

// Refresh POST /api/v1/auth/refresh （读 refresh cookie，轮换续期）
func Refresh(w http.ResponseWriter, r *http.Request) {
	if sessionSvc == nil {
		response.Fail(w, errorx.ErrInternal)
		return
	}
	rt := readRefreshCookie(r)
	if rt == "" {
		response.Fail(w, errorx.ErrUnauthorized.WithMsg("未登录"))
		return
	}
	pair, err := sessionSvc.Refresh(r.Context(), rt, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		clearRefreshCookie(w)
		response.Fail(w, err)
		return
	}
	setRefreshCookie(w, pair.Refresh)
	response.OK(w, openapi.TokenData{AccessToken: pair.Access})
}

// Logout POST /api/v1/auth/logout （吊销会话、清 cookie）
func Logout(w http.ResponseWriter, r *http.Request) {
	if sessionSvc != nil {
		if rt := readRefreshCookie(r); rt != "" {
			_ = sessionSvc.Revoke(r.Context(), rt)
		}
	}
	clearRefreshCookie(w)
	response.OK(w, nil)
}

// GetMe GET /api/v1/users/me （需 Auth 中间件，userID 来自 context）
func GetMe(w http.ResponseWriter, r *http.Request) {
	uid := ctxkit.UserID(r.Context())
	if uid == "" {
		response.Fail(w, errorx.ErrUnauthorized)
		return
	}
	if svc == nil {
		response.Fail(w, errorx.ErrInternal)
		return
	}
	u, err := svc.GetByID(r.Context(), uid)
	if err != nil {
		response.Fail(w, err)
		return
	}
	response.OK(w, openapi.MeData{User: toModel(u)})
}
