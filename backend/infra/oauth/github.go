// Package oauth 第三方登录的基础设施实现（外部 HTTP 调用）。
// 现有 GitHub；以后加 Google/Apple 各写一个客户端即可（易扩展）。
package oauth

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// Profile 第三方用户的归一化资料（各 provider 客户端都产出这个形状）。
type Profile struct {
	ID     string
	Login  string
	Name   string
	Email  string
	Avatar string
}

// GitHub OAuth 客户端：拼授权 URL + 用 code 换 token + 拉用户资料。
// client_secret 只存在后端，不下发前端（安全）。
type GitHub struct {
	clientID     string
	clientSecret string
	redirectURL  string
	tokenURL     string // 换 token 端点（测试可覆盖指向 httptest）
	apiBase      string // GitHub API 根（测试可覆盖）
	hc           *http.Client
}

func NewGitHub(clientID, clientSecret, redirectURL string) *GitHub {
	return &GitHub{
		clientID:     clientID,
		clientSecret: clientSecret,
		redirectURL:  redirectURL,
		tokenURL:     "https://github.com/login/oauth/access_token",
		apiBase:      "https://api.github.com",
		hc:           &http.Client{Timeout: 10 * time.Second}, // 超时，避免外部依赖拖垮请求
	}
}

// Configured 未配置 client 时功能优雅关闭。
func (g *GitHub) Configured() bool { return g.clientID != "" && g.clientSecret != "" }

// AuthorizeURL 浏览器要跳转去的 GitHub 授权地址。
func (g *GitHub) AuthorizeURL(state string) string {
	q := url.Values{}
	q.Set("client_id", g.clientID)
	q.Set("redirect_uri", g.redirectURL)
	q.Set("scope", "read:user user:email")
	q.Set("state", state)
	q.Set("allow_signup", "true")
	return "https://github.com/login/oauth/authorize?" + q.Encode()
}

// Exchange 用 code 换 access_token，再拉取用户资料。
func (g *GitHub) Exchange(ctx context.Context, code string) (*Profile, error) {
	token, err := g.exchangeToken(ctx, code)
	if err != nil {
		return nil, err
	}
	return g.fetchUser(ctx, token)
}

func (g *GitHub) exchangeToken(ctx context.Context, code string) (string, error) {
	form := url.Values{}
	form.Set("client_id", g.clientID)
	form.Set("client_secret", g.clientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", g.redirectURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		g.tokenURL, strings.NewReader(form.Encode()))
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := g.hc.Do(req)
	if err != nil {
		return "", err
	}
	defer func() { _ = resp.Body.Close() }()
	var out struct {
		AccessToken string `json:"access_token"`
		Error       string `json:"error"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		return "", err
	}
	if out.AccessToken == "" {
		return "", fmt.Errorf("github oauth: 换取 token 失败: %s", out.Error)
	}
	return out.AccessToken, nil
}

func (g *GitHub) fetchUser(ctx context.Context, token string) (*Profile, error) {
	body, err := g.apiGet(ctx, token, g.apiBase+"/user")
	if err != nil {
		return nil, err
	}
	var u struct {
		ID     int64  `json:"id"`
		Login  string `json:"login"`
		Name   string `json:"name"`
		Email  string `json:"email"`
		Avatar string `json:"avatar_url"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return nil, err
	}
	email := u.Email
	if email == "" {
		email = g.primaryEmail(ctx, token) // 邮箱设私有时尽力补一次
	}
	return &Profile{
		ID:     strconv.FormatInt(u.ID, 10),
		Login:  u.Login,
		Name:   u.Name,
		Email:  email,
		Avatar: u.Avatar,
	}, nil
}

func (g *GitHub) primaryEmail(ctx context.Context, token string) string {
	body, err := g.apiGet(ctx, token, g.apiBase+"/user/emails")
	if err != nil {
		return ""
	}
	var emails []struct {
		Email    string `json:"email"`
		Primary  bool   `json:"primary"`
		Verified bool   `json:"verified"`
	}
	if json.Unmarshal(body, &emails) != nil {
		return ""
	}
	for _, e := range emails {
		if e.Primary && e.Verified {
			return e.Email
		}
	}
	return ""
}

func (g *GitHub) apiGet(ctx context.Context, token, u string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	resp, err := g.hc.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("github api %s: status %d", u, resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}
