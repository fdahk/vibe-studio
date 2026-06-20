package persistence

import (
	"context"
	"errors"
	"time"

	"gorm.io/gorm"

	domain "vibe-studio/backend/domain/user"
	"vibe-studio/backend/pkg/errorx"
)

var _ domain.Repository = (*Repo)(nil)

// userPO 用户表（身份主体）持久化对象。phone 仅作 profile 检索（非唯一）；
// 手机号唯一性由 identities(provider='phone', provider_uid) 的唯一索引保证。
type userPO struct {
	ID        string `gorm:"primaryKey;size:64"`
	Username  string `gorm:"size:64;uniqueIndex;not null"`
	Email     string `gorm:"size:128"`
	Phone     string `gorm:"size:32;index"`
	Nickname  string `gorm:"size:64"`
	Avatar    string `gorm:"size:255"`
	Status    string `gorm:"size:16"`
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (userPO) TableName() string { return "users" }

// identityPO 登录身份表：(provider, provider_uid) 唯一。
type identityPO struct {
	ID          string `gorm:"primaryKey;size:64"`
	UserID      string `gorm:"size:64;not null;index"`
	Provider    string `gorm:"size:32;not null;uniqueIndex:uk_provider_uid,priority:1"`
	ProviderUID string `gorm:"size:191;not null;uniqueIndex:uk_provider_uid,priority:2"`
	Secret      string `gorm:"size:255"`
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

func (identityPO) TableName() string { return "identities" }

type Repo struct{ db *gorm.DB }

func NewRepo(db *gorm.DB) *Repo { return &Repo{db: db} }

func (r *Repo) GetUserByID(ctx context.Context, id string) (*domain.User, error) {
	var po userPO
	err := r.db.WithContext(ctx).First(&po, "id = ?", id).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return toUser(po), nil
}

func (r *Repo) GetIdentity(ctx context.Context, provider, providerUID string) (*domain.Identity, error) {
	var po identityPO
	err := r.db.WithContext(ctx).First(&po, "provider = ? AND provider_uid = ?", provider, providerUID).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, errorx.ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	return &domain.Identity{
		ID: po.ID, UserID: po.UserID, Provider: po.Provider,
		ProviderUID: po.ProviderUID, Secret: po.Secret, CreatedAt: po.CreatedAt,
	}, nil
}

// CreateAccount 事务内创建 user + 首个 identity，二者全有或全无。
func (r *Repo) CreateAccount(ctx context.Context, u *domain.User, id *domain.Identity) error {
	upo := userPO{
		ID: u.ID, Username: u.Username, Email: u.Email, Phone: u.Phone,
		Nickname: u.Nickname, Avatar: u.Avatar, Status: u.Status,
	}
	ipo := identityPO{
		ID: id.ID, UserID: id.UserID, Provider: id.Provider,
		ProviderUID: id.ProviderUID, Secret: id.Secret,
	}
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Create(&upo).Error; err != nil {
			return err
		}
		return tx.Create(&ipo).Error
	})
	if err != nil {
		return err
	}
	u.CreatedAt = upo.CreatedAt
	id.CreatedAt = ipo.CreatedAt
	return nil
}

func toUser(po userPO) *domain.User {
	return &domain.User{
		ID: po.ID, Username: po.Username, Email: po.Email, Phone: po.Phone,
		Nickname: po.Nickname, Avatar: po.Avatar, Status: po.Status, CreatedAt: po.CreatedAt,
	}
}
