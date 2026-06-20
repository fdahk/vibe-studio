package user

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	userapp "vibe-studio/backend/application/user"
	domain "vibe-studio/backend/domain/user"
	"vibe-studio/backend/pkg/auth"
	"vibe-studio/backend/pkg/errorx"
)

// ---- 内存假实现：用真实 Service/SessionService + 假仓储，测 handler 而不依赖 DB ----

type memRepo struct {
	users      map[string]*domain.User
	identities map[string]*domain.Identity
}

func newMemRepo() *memRepo {
	return &memRepo{users: map[string]*domain.User{}, identities: map[string]*domain.Identity{}}
}
func (m *memRepo) GetUserByID(_ context.Context, id string) (*domain.User, error) {
	if u, ok := m.users[id]; ok {
		return u, nil
	}
	return nil, errorx.ErrNotFound
}
func (m *memRepo) GetIdentity(_ context.Context, p, uid string) (*domain.Identity, error) {
	if i, ok := m.identities[p+":"+uid]; ok {
		return i, nil
	}
	return nil, errorx.ErrNotFound
}
func (m *memRepo) CreateAccount(_ context.Context, u *domain.User, id *domain.Identity) error {
	m.users[u.ID] = u
	m.identities[id.Provider+":"+id.ProviderUID] = id
	return nil
}

type memSessionRepo struct{ all []*domain.Session }

func (m *memSessionRepo) Create(_ context.Context, s *domain.Session) error {
	m.all = append(m.all, s)
	return nil
}
func (m *memSessionRepo) FindByTokenHash(_ context.Context, h string) (*domain.Session, error) {
	for _, s := range m.all {
		if s.TokenHash == h {
			return s, nil
		}
	}
	return nil, errorx.ErrNotFound
}
func (m *memSessionRepo) FindByPrevTokenHash(_ context.Context, h string) (*domain.Session, error) {
	for _, s := range m.all {
		if s.PrevTokenHash == h {
			return s, nil
		}
	}
	return nil, errorx.ErrNotFound
}
func (m *memSessionRepo) Update(_ context.Context, _ *domain.Session) error { return nil }

type noopSMS struct{}

func (noopSMS) Send(context.Context, string, string) error { return nil }

type noopCodes struct{}

func (noopCodes) Issue(context.Context, string) (string, error) { return "123456", nil }
func (noopCodes) Verify(context.Context, string, string) error  { return nil }

func setupHandlers() {
	svc := userapp.NewService(newMemRepo(), noopSMS{}, noopCodes{})
	sessionSvc := userapp.NewSessionService(&memSessionRepo{}, auth.NewJWT("secret", time.Hour), time.Hour)
	SetService(svc)
	SetSession(sessionSvc, CookieConfig{Name: "vibe_refresh", MaxAge: 3600})
}

func doJSON(h http.HandlerFunc, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func findCookie(rec *httptest.ResponseRecorder, name string) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == name {
			return c
		}
	}
	return nil
}

func TestHandlerRegisterSetsCookieAndAccess(t *testing.T) {
	setupHandlers()
	rec := doJSON(Register, `{"username":"alice","password":"pw123456"}`)
	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"access_token"`)
	c := findCookie(rec, "vibe_refresh")
	require.NotNil(t, c, "应种 refresh cookie")
	assert.True(t, c.HttpOnly)
	assert.NotEmpty(t, c.Value)
}

func TestHandlerLoginWrongPasswordNoCookie(t *testing.T) {
	setupHandlers()
	doJSON(Register, `{"username":"bob","password":"pw123456"}`)
	rec := doJSON(Login, `{"username":"bob","password":"wrong"}`)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
	assert.Nil(t, findCookie(rec, "vibe_refresh"), "登录失败不应种 cookie")
}

func TestHandlerRefreshRotatesCookie(t *testing.T) {
	setupHandlers()
	reg := doJSON(Register, `{"username":"carol","password":"pw123456"}`)
	rt := findCookie(reg, "vibe_refresh").Value

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil)
	req.AddCookie(&http.Cookie{Name: "vibe_refresh", Value: rt})
	rec := httptest.NewRecorder()
	Refresh(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	assert.Contains(t, rec.Body.String(), `"access_token"`)
	nc := findCookie(rec, "vibe_refresh")
	require.NotNil(t, nc)
	assert.NotEqual(t, rt, nc.Value, "refresh cookie 应轮换")
}

func TestHandlerRefreshWithoutCookie(t *testing.T) {
	setupHandlers()
	rec := httptest.NewRecorder()
	Refresh(rec, httptest.NewRequest(http.MethodPost, "/api/v1/auth/refresh", nil))
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestHandlerLogoutClearsCookie(t *testing.T) {
	setupHandlers()
	reg := doJSON(Register, `{"username":"dave","password":"pw123456"}`)
	rt := findCookie(reg, "vibe_refresh").Value

	req := httptest.NewRequest(http.MethodPost, "/api/v1/auth/logout", nil)
	req.AddCookie(&http.Cookie{Name: "vibe_refresh", Value: rt})
	rec := httptest.NewRecorder()
	Logout(rec, req)

	assert.Equal(t, http.StatusOK, rec.Code)
	c := findCookie(rec, "vibe_refresh")
	require.NotNil(t, c)
	assert.True(t, c.MaxAge < 0, "登出应清 refresh cookie")
}
