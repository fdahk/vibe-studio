package user

import "context"

// Repository 用户+身份的出站端口（实现见 infra/persistence）。
// 约定：查不到返回 errorx.ErrNotFound，由应用层翻译成合适的对外错误。
type Repository interface {
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetIdentity(ctx context.Context, provider, providerUID string) (*Identity, error)
	// CreateAccount 在一个事务内创建 user 及其首个 identity，保证两者一致性。
	CreateAccount(ctx context.Context, u *User, id *Identity) error
}
