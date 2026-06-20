package userapp

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	domain "vibe-studio/backend/domain/user"
	"vibe-studio/backend/pkg/auth"
	"vibe-studio/backend/pkg/errorx"
)

// fakeSessionRepo 内存实现，指针共享 → Update 无需复制即生效。
type fakeSessionRepo struct{ all []*domain.Session }

func (f *fakeSessionRepo) Create(_ context.Context, s *domain.Session) error {
	f.all = append(f.all, s)
	return nil
}

func (f *fakeSessionRepo) FindByTokenHash(_ context.Context, hash string) (*domain.Session, error) {
	for _, s := range f.all {
		if s.TokenHash == hash {
			return s, nil
		}
	}
	return nil, errorx.ErrNotFound
}

func (f *fakeSessionRepo) FindByPrevTokenHash(_ context.Context, hash string) (*domain.Session, error) {
	for _, s := range f.all {
		if s.PrevTokenHash == hash {
			return s, nil
		}
	}
	return nil, errorx.ErrNotFound
}

func (f *fakeSessionRepo) Update(_ context.Context, _ *domain.Session) error { return nil }

func newSessionService() (*SessionService, *fakeSessionRepo) {
	repo := &fakeSessionRepo{}
	return NewSessionService(repo, auth.NewJWT("test-secret", time.Hour), 24*time.Hour), repo
}

func TestSessionIssue(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSessionService()

	pair, err := svc.Issue(ctx, "user-1", "agent", "1.2.3.4")
	require.NoError(t, err)
	assert.NotEmpty(t, pair.Access)
	assert.NotEmpty(t, pair.Refresh)

	require.Len(t, repo.all, 1)
	s := repo.all[0]
	assert.Equal(t, "user-1", s.UserID)
	assert.Equal(t, auth.HashToken(pair.Refresh), s.TokenHash, "库里存 refresh 的哈希")
	assert.False(t, s.IsRevoked())
	assert.True(t, s.ExpiresAt.After(time.Now()))
}

func TestSessionRefreshRotates(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSessionService()
	first, _ := svc.Issue(ctx, "user-1", "a", "ip")

	second, err := svc.Refresh(ctx, first.Refresh, "a", "ip")
	require.NoError(t, err)
	assert.NotEmpty(t, second.Access)
	assert.NotEqual(t, first.Refresh, second.Refresh, "refresh 应轮换")

	s := repo.all[0]
	assert.Equal(t, auth.HashToken(second.Refresh), s.TokenHash)
	assert.Equal(t, auth.HashToken(first.Refresh), s.PrevTokenHash, "旧 refresh 进 prev")
}

func TestSessionRefreshRejectsUnknown(t *testing.T) {
	ctx := context.Background()
	svc, _ := newSessionService()
	_, err := svc.Refresh(ctx, "nonexistent", "a", "ip")
	assert.Error(t, err)
}

func TestSessionRefreshReuseRevokesSession(t *testing.T) {
	ctx := context.Background()
	svc, repo := newSessionService()
	first, _ := svc.Issue(ctx, "user-1", "a", "ip")
	second, _ := svc.Refresh(ctx, first.Refresh, "a", "ip") // first 现在是 prev

	// 重放已轮换的 first → 复用检测 → 吊销整条会话
	_, err := svc.Refresh(ctx, first.Refresh, "a", "ip")
	assert.Error(t, err)
	assert.True(t, repo.all[0].IsRevoked(), "复用旧 token 应吊销会话")

	// 之后连合法的 second 也用不了
	_, err = svc.Refresh(ctx, second.Refresh, "a", "ip")
	assert.Error(t, err, "会话已吊销")
}

func TestSessionRefreshRejectsExpired(t *testing.T) {
	ctx := context.Background()
	repo := &fakeSessionRepo{}
	svc := NewSessionService(repo, auth.NewJWT("s", time.Hour), -time.Hour) // 负 TTL → 一签发即过期
	first, _ := svc.Issue(ctx, "user-1", "a", "ip")
	_, err := svc.Refresh(ctx, first.Refresh, "a", "ip")
	assert.Error(t, err, "过期会话拒绝")
}

func TestSessionRevokeThenRefreshFails(t *testing.T) {
	ctx := context.Background()
	svc, _ := newSessionService()
	first, _ := svc.Issue(ctx, "user-1", "a", "ip")
	require.NoError(t, svc.Revoke(ctx, first.Refresh))
	_, err := svc.Refresh(ctx, first.Refresh, "a", "ip")
	assert.Error(t, err, "登出后 refresh 失败")
}
