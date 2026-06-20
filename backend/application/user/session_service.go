package userapp

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"

	domain "vibe-studio/backend/domain/user"
	"vibe-studio/backend/pkg/auth"
	"vibe-studio/backend/pkg/errorx"
)

// TokenPair 一次签发的访问/刷新令牌对。
type TokenPair struct {
	Access  string // JWT，短时，放响应体
	Refresh string // 不透明随机串，放 HttpOnly cookie
}

// SessionService 管理登录会话：签发、轮换（含复用检测）、吊销。
type SessionService struct {
	repo       domain.SessionRepository
	jwt        *auth.JWT
	refreshTTL time.Duration
}

func NewSessionService(repo domain.SessionRepository, jwt *auth.JWT, refreshTTL time.Duration) *SessionService {
	return &SessionService{repo: repo, jwt: jwt, refreshTTL: refreshTTL}
}

// Issue 登录成功后建会话并签发 access+refresh。
func (s *SessionService) Issue(ctx context.Context, userID, userAgent, ip string) (*TokenPair, error) {
	refresh, err := auth.GenerateRefreshToken()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	sess := &domain.Session{
		ID:         uuid.NewString(),
		UserID:     userID,
		TokenHash:  auth.HashToken(refresh),
		UserAgent:  userAgent,
		IP:         ip,
		ExpiresAt:  now.Add(s.refreshTTL),
		LastUsedAt: now,
	}
	if err := s.repo.Create(ctx, sess); err != nil {
		return nil, err
	}
	return s.pair(sess.UserID, refresh)
}

// Refresh 轮换：校验当前 refresh→换发新 refresh（旧的进 prev）。
// 若来的是已轮换的旧 refresh（复用/泄露）→ 吊销整条会话。
func (s *SessionService) Refresh(ctx context.Context, refreshToken, _, _ string) (*TokenPair, error) {
	hash := auth.HashToken(refreshToken)

	sess, err := s.repo.FindByTokenHash(ctx, hash)
	if err == nil {
		if sess.IsRevoked() || sess.IsExpired(time.Now()) {
			return nil, errSessionInvalid()
		}
		newRefresh, err := auth.GenerateRefreshToken()
		if err != nil {
			return nil, err
		}
		now := time.Now()
		sess.PrevTokenHash = sess.TokenHash
		sess.TokenHash = auth.HashToken(newRefresh)
		sess.LastUsedAt = now
		if err := s.repo.Update(ctx, sess); err != nil {
			return nil, err
		}
		return s.pair(sess.UserID, newRefresh)
	}
	if !errors.Is(err, errorx.ErrNotFound) {
		return nil, err
	}

	// 复用检测：命中已被轮换的旧 token → 判泄露 → 吊销整条会话。
	if reused, err2 := s.repo.FindByPrevTokenHash(ctx, hash); err2 == nil {
		_ = s.revoke(ctx, reused)
	}
	return nil, errSessionInvalid()
}

// Revoke 登出：吊销 refresh 对应的会话（幂等）。
func (s *SessionService) Revoke(ctx context.Context, refreshToken string) error {
	sess, err := s.repo.FindByTokenHash(ctx, auth.HashToken(refreshToken))
	if errors.Is(err, errorx.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	return s.revoke(ctx, sess)
}

func (s *SessionService) revoke(ctx context.Context, sess *domain.Session) error {
	now := time.Now()
	sess.RevokedAt = &now
	return s.repo.Update(ctx, sess)
}

func (s *SessionService) pair(userID, refresh string) (*TokenPair, error) {
	access, err := s.jwt.Issue(userID)
	if err != nil {
		return nil, err
	}
	return &TokenPair{Access: access, Refresh: refresh}, nil
}

func errSessionInvalid() error {
	return errorx.ErrUnauthorized.WithMsg("会话已失效，请重新登录")
}
