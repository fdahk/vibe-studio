package userapp

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "vibe-studio/backend/domain/user"
	"vibe-studio/backend/pkg/auth"
	"vibe-studio/backend/pkg/errorx"
)

// fakeRepo 是 domain.Repository 的内存实现：换个假实现就能单测业务编排，不依赖 DB。
type fakeRepo struct {
	users      map[string]*domain.User     // by id
	identities map[string]*domain.Identity // by "provider:uid"
}

func newFakeRepo() *fakeRepo {
	return &fakeRepo{users: map[string]*domain.User{}, identities: map[string]*domain.Identity{}}
}

func (f *fakeRepo) GetUserByID(_ context.Context, id string) (*domain.User, error) {
	if u, ok := f.users[id]; ok {
		return u, nil
	}
	return nil, errorx.ErrNotFound
}

func (f *fakeRepo) GetIdentity(_ context.Context, provider, uid string) (*domain.Identity, error) {
	if i, ok := f.identities[provider+":"+uid]; ok {
		return i, nil
	}
	return nil, errorx.ErrNotFound
}

func (f *fakeRepo) CreateAccount(_ context.Context, u *domain.User, id *domain.Identity) error {
	f.users[u.ID] = u
	f.identities[id.Provider+":"+id.ProviderUID] = id
	return nil
}

// fakeSMS 记录最后一次下发的验证码。
type fakeSMS struct{ last string }

func (f *fakeSMS) Send(_ context.Context, _, code string) error { f.last = code; return nil }

// fakeCodeStore 固定验证码 123456，便于断言。
type fakeCodeStore struct{}

func (fakeCodeStore) Issue(_ context.Context, _ string) (string, error) { return "123456", nil }
func (fakeCodeStore) Verify(_ context.Context, _, code string) error {
	if code == "123456" {
		return nil
	}
	return errorx.ErrSMSCodeInvalid
}

func newService() (*Service, *fakeRepo) {
	repo := newFakeRepo()
	return NewService(repo, &fakeSMS{}, fakeCodeStore{}), repo
}

func TestRegister(t *testing.T) {
	ctx := context.Background()
	svc, repo := newService()

	u, err := svc.Register(ctx, "alice", "pw123456", "a@x.com")
	require.NoError(t, err)
	assert.Equal(t, "alice", u.Username)

	// 落库的是哈希而非明文，且能用原密码校验。
	id := repo.identities[domain.ProviderLocal+":alice"]
	require.NotNil(t, id)
	assert.NotEqual(t, "pw123456", id.Secret)
	assert.True(t, auth.VerifyPassword(id.Secret, "pw123456"))

	_, err = svc.Register(ctx, "alice", "other", "")
	assert.Error(t, err, "重名应冲突")

	_, err = svc.Register(ctx, "", "pw", "")
	assert.Error(t, err, "缺用户名/密码应报错")
}

func TestLogin(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService()
	_, err := svc.Register(ctx, "bob", "pw123456", "")
	require.NoError(t, err)

	u, err := svc.Login(ctx, "bob", "pw123456")
	require.NoError(t, err)
	assert.Equal(t, "bob", u.Username)

	_, err = svc.Login(ctx, "bob", "wrong")
	assert.Error(t, err, "错误密码应失败")

	_, err = svc.Login(ctx, "ghost", "pw")
	assert.Error(t, err, "不存在的用户应失败")
}

func TestLoginByPhone(t *testing.T) {
	ctx := context.Background()
	svc, _ := newService()
	const phone = "13800138000"

	require.NoError(t, svc.SendSMSCode(ctx, phone))
	assert.Error(t, svc.SendSMSCode(ctx, "123"), "非法手机号应报错")

	_, err := svc.LoginByPhone(ctx, phone, "000000")
	assert.Error(t, err, "错误验证码应失败")

	// 正确验证码 → 新号自动注册。
	u, err := svc.LoginByPhone(ctx, phone, "123456")
	require.NoError(t, err)
	assert.Equal(t, phone, u.Phone)

	// 再次登录 → 命中同一用户，不重复建号。
	u2, err := svc.LoginByPhone(ctx, phone, "123456")
	require.NoError(t, err)
	assert.Equal(t, u.ID, u2.ID)
}

func TestLoginByOAuth(t *testing.T) {
	ctx := context.Background()
	svc, repo := newService()

	// 新第三方账号 → 自动注册。
	u, err := svc.LoginByOAuth(ctx, domain.ProviderGitHub, "g1", "g@x.com", "Octo", "http://avatar")
	require.NoError(t, err)
	assert.Equal(t, "github_g1", u.Username)
	assert.Equal(t, "Octo", u.Nickname)
	assert.Equal(t, "g@x.com", u.Email)
	require.NotNil(t, repo.identities[domain.ProviderGitHub+":g1"], "建立 github identity")

	// 同一第三方账号再次登录 → 命中同一用户，不重复建号。
	u2, err := svc.LoginByOAuth(ctx, domain.ProviderGitHub, "g1", "", "", "")
	require.NoError(t, err)
	assert.Equal(t, u.ID, u2.ID)

	// 昵称为空 → 兜底非空。
	u3, err := svc.LoginByOAuth(ctx, domain.ProviderGitHub, "g2", "", "", "")
	require.NoError(t, err)
	assert.NotEmpty(t, u3.Nickname)
	assert.NotEqual(t, u.ID, u3.ID, "不同第三方 uid 是不同用户")
}
