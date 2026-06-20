package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	domain "vibe-studio/backend/domain/user"
	"vibe-studio/backend/pkg/errorx"
)

var _ domain.SessionRepository = (*SessionRepo)(nil)

type sessionPO struct {
	ID            string `gorm:"primaryKey;size:64"`
	UserID        string `gorm:"size:64;not null;index"`
	TokenHash     string `gorm:"size:64;not null;uniqueIndex"`
	PrevTokenHash string `gorm:"size:64;index"`
	UserAgent     string `gorm:"size:255"`
	IP            string `gorm:"size:64"`
	ExpiresAt     time.Time
	RevokedAt     *time.Time
	LastUsedAt    time.Time
	CreatedAt     time.Time
	UpdatedAt     time.Time
}

func (sessionPO) TableName() string { return "sessions" }

type SessionRepo struct{ db *gorm.DB }

func NewSessionRepo(db *gorm.DB) *SessionRepo { return &SessionRepo{db: db} }

func (r *SessionRepo) Create(ctx context.Context, s *domain.Session) error {
	po := toSessionPO(s)
	if err := r.db.WithContext(ctx).Create(&po).Error; err != nil {
		return err
	}
	s.CreatedAt = po.CreatedAt
	return nil
}

func (r *SessionRepo) FindByTokenHash(ctx context.Context, hash string) (*domain.Session, error) {
	return r.findBy(ctx, "token_hash = ?", hash)
}

func (r *SessionRepo) FindByPrevTokenHash(ctx context.Context, hash string) (*domain.Session, error) {
	return r.findBy(ctx, "prev_token_hash = ?", hash)
}

func (r *SessionRepo) findBy(ctx context.Context, cond, arg string) (*domain.Session, error) {
	var po sessionPO
	err := r.db.WithContext(ctx).First(&po, cond, arg).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toSession(po), nil
}

// Update 只更新可变列（轮换/吊销），不碰 created_at（用 Save 会把它清成零值越界）。
func (r *SessionRepo) Update(ctx context.Context, s *domain.Session) error {
	return r.db.WithContext(ctx).Model(&sessionPO{ID: s.ID}).Updates(map[string]any{
		"token_hash":      s.TokenHash,
		"prev_token_hash": s.PrevTokenHash,
		"last_used_at":    s.LastUsedAt,
		"revoked_at":      s.RevokedAt,
		"updated_at":      time.Now(),
	}).Error
}

func toSessionPO(s *domain.Session) sessionPO {
	return sessionPO{
		ID: s.ID, UserID: s.UserID, TokenHash: s.TokenHash, PrevTokenHash: s.PrevTokenHash,
		UserAgent: s.UserAgent, IP: s.IP, ExpiresAt: s.ExpiresAt,
		RevokedAt: s.RevokedAt, LastUsedAt: s.LastUsedAt,
	}
}

func toSession(po sessionPO) *domain.Session {
	return &domain.Session{
		ID: po.ID, UserID: po.UserID, TokenHash: po.TokenHash, PrevTokenHash: po.PrevTokenHash,
		UserAgent: po.UserAgent, IP: po.IP, ExpiresAt: po.ExpiresAt,
		RevokedAt: po.RevokedAt, LastUsedAt: po.LastUsedAt, CreatedAt: po.CreatedAt,
	}
}
