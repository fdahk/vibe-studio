package user

import "time"

// Provider 登录方式（业界标准多 provider 鉴权：一个用户可绑定多个登录身份）。
const (
	ProviderLocal  = "local"  // 用户名 + 密码
	ProviderPhone  = "phone"  // 手机号 + 短信验证码
	ProviderGitHub = "github" // GitHub OAuth
	ProviderGoogle = "google" // 预留：Google OAuth
	ProviderApple  = "apple"  // 预留：Apple Sign in
)

// StatusActive 账号正常状态。
const StatusActive = "active"

// User 用户聚合根（身份主体 / principal）。**不含任何登录凭证**——凭证在 Identity。
type User struct {
	ID        string
	Username  string
	Email     string
	Phone     string
	Nickname  string
	Avatar    string
	Status    string
	CreatedAt time.Time
}

// Identity 登录身份/凭证（users 1—N identities，业界标准联合身份模型）。
// 一条 identity = 一种登录方式，(Provider, ProviderUID) 全局唯一：
//   - local：ProviderUID=用户名，Secret=bcrypt 密码哈希
//   - phone：ProviderUID=手机号，Secret 为空（短信验证码核验，不存密码）
//   - google/apple：ProviderUID=第三方 sub，Secret 预留
type Identity struct {
	ID          string
	UserID      string
	Provider    string
	ProviderUID string
	Secret      string
	CreatedAt   time.Time
}
