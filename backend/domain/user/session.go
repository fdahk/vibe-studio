package user

import (
	"context"
	"time"
)

// Session 登录会话（一台设备一条）。refresh token 只存哈希；轮换时旧哈希进 PrevTokenHash 供复用检测。
type Session struct {
	ID            string
	UserID        string
	TokenHash     string
	PrevTokenHash string
	UserAgent     string
	IP            string
	ExpiresAt     time.Time
	RevokedAt     *time.Time // nil = 未吊销
	LastUsedAt    time.Time
	CreatedAt     time.Time
}

func (s *Session) IsRevoked() bool            { return s.RevokedAt != nil }
func (s *Session) IsExpired(t time.Time) bool { return t.After(s.ExpiresAt) }

// SessionRepository 会话出站端口（实现见 infra/persistence）。查不到返回 errorx.ErrNotFound。
type SessionRepository interface {
	Create(ctx context.Context, s *Session) error
	FindByTokenHash(ctx context.Context, hash string) (*Session, error)
	FindByPrevTokenHash(ctx context.Context, hash string) (*Session, error)
	Update(ctx context.Context, s *Session) error
}
