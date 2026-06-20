package user

import (
	"net/http"

	domain "vibe-studio/backend/domain/user"
	"vibe-studio/backend/infra/oauth"
	"vibe-studio/backend/pkg/errorx"
	"vibe-studio/backend/pkg/response"
)

// OAuth 相关依赖由组合根注入。
var (
	githubClient *oauth.GitHub
	stateStore   *oauth.StateStore
	frontendURL  string
)

func SetOAuth(gh *oauth.GitHub, ss *oauth.StateStore, feURL string) {
	githubClient = gh
	stateStore = ss
	frontendURL = feURL
}

// OAuthGitHubLogin GET /api/v1/auth/oauth/github → 带 state 跳转 GitHub 授权页。
func OAuthGitHubLogin(w http.ResponseWriter, r *http.Request) {
	if githubClient == nil || !githubClient.Configured() {
		response.Fail(w, errorx.ErrInternal.WithMsg("GitHub 登录未配置"))
		return
	}
	state, err := stateStore.Issue(r.Context())
	if err != nil {
		response.Fail(w, errorx.ErrInternal.Wrap(err))
		return
	}
	http.Redirect(w, r, githubClient.AuthorizeURL(state), http.StatusFound)
}

// OAuthGitHubCallback GET /api/v1/auth/oauth/github/callback?code=&state=
// 校验 state → 换 token 拉用户 → find-or-create → 跳回前端(token 放 fragment)。
func OAuthGitHubCallback(w http.ResponseWriter, r *http.Request) {
	if githubClient == nil || svc == nil {
		response.Fail(w, errorx.ErrInternal)
		return
	}
	q := r.URL.Query()
	if !stateStore.Consume(r.Context(), q.Get("state")) {
		response.Fail(w, errorx.ErrUnauthorized.WithMsg("state 无效或已过期"))
		return
	}
	profile, err := githubClient.Exchange(r.Context(), q.Get("code"))
	if err != nil {
		response.Fail(w, errorx.ErrUnauthorized.Wrap(err))
		return
	}
	u, err := svc.LoginByOAuth(r.Context(), domain.ProviderGitHub,
		profile.ID, profile.Email, firstNonEmpty(profile.Name, profile.Login), profile.Avatar)
	if err != nil {
		response.Fail(w, err)
		return
	}
	if sessionSvc == nil {
		response.Fail(w, errorx.ErrInternal)
		return
	}
	pair, err := sessionSvc.Issue(r.Context(), u.ID, r.UserAgent(), r.RemoteAddr)
	if err != nil {
		response.Fail(w, errorx.ErrInternal.Wrap(err))
		return
	}
	// 只种 refresh cookie，URL 不带 token；前端落地后调 /refresh 拿 access。
	setRefreshCookie(w, pair.Refresh)
	http.Redirect(w, r, frontendURL+"/oauth/callback", http.StatusFound)
}

func firstNonEmpty(a, b string) string {
	if a != "" {
		return a
	}
	return b
}
