package userapp

import (
	"context"
	"errors"
	"regexp"

	"github.com/google/uuid"

	domain "vibe-studio/backend/domain/user"
	"vibe-studio/backend/pkg/auth"
	"vibe-studio/backend/pkg/errorx"
)

// SMSSender 短信发送出站端口（实现见 infra/sms）。
type SMSSender interface {
	Send(ctx context.Context, phone, code string) error
}

// CodeStore 验证码存储/校验出站端口（实现见 infra/sms，基于 Redis）。
type CodeStore interface {
	Issue(ctx context.Context, phone string) (code string, err error)
	Verify(ctx context.Context, phone, code string) error
}

// cnPhone 中国大陆手机号。以后支持国际号码时这里放宽即可。
var cnPhone = regexp.MustCompile(`^1[3-9]\d{9}$`)

// Service 用户应用服务：负责**认证**（多 provider 登录/注册），返回认证出的用户主体；
// access/refresh 的签发交给 SessionService（由 handler 用请求上下文调用）。
type Service struct {
	repo  domain.Repository
	sms   SMSSender
	codes CodeStore
}

func NewService(repo domain.Repository, sms SMSSender, codes CodeStore) *Service {
	return &Service{repo: repo, sms: sms, codes: codes}
}

// Register 用户名密码注册：查重 → 哈希 → 建 user+local identity。
func (s *Service) Register(ctx context.Context, username, password, email string) (*domain.User, error) {
	if username == "" || password == "" {
		return nil, errorx.ErrBadRequest.WithMsg("用户名和密码必填")
	}
	if _, err := s.repo.GetIdentity(ctx, domain.ProviderLocal, username); err == nil {
		return nil, errorx.ErrConflict.WithMsg("用户名已存在")
	}
	hash, err := auth.HashPassword(password)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		ID: uuid.NewString(), Username: username, Email: email,
		Nickname: username, Status: domain.StatusActive,
	}
	id := &domain.Identity{
		ID: uuid.NewString(), UserID: u.ID,
		Provider: domain.ProviderLocal, ProviderUID: username, Secret: hash,
	}
	if err := s.repo.CreateAccount(ctx, u, id); err != nil {
		return nil, err
	}
	return u, nil
}

// Login 用户名密码登录（失败统一报"用户名或密码错误"，不泄露账号是否存在）。
func (s *Service) Login(ctx context.Context, username, password string) (*domain.User, error) {
	id, err := s.repo.GetIdentity(ctx, domain.ProviderLocal, username)
	if err != nil || !auth.VerifyPassword(id.Secret, password) {
		return nil, errorx.ErrUnauthorized.WithMsg("用户名或密码错误")
	}
	return s.repo.GetUserByID(ctx, id.UserID)
}

// SendSMSCode 校验手机号 → 生成验证码（带频控）→ 下发。
func (s *Service) SendSMSCode(ctx context.Context, phone string) error {
	if !cnPhone.MatchString(phone) {
		return errorx.ErrPhoneInvalid
	}
	code, err := s.codes.Issue(ctx, phone)
	if err != nil {
		return err
	}
	if err := s.sms.Send(ctx, phone, code); err != nil {
		return errorx.ErrInternal.Wrap(err)
	}
	return nil
}

// LoginByPhone 手机号验证码登录：校验码 → 命中 phone identity 则登录，否则自动注册。
func (s *Service) LoginByPhone(ctx context.Context, phone, code string) (*domain.User, error) {
	if !cnPhone.MatchString(phone) {
		return nil, errorx.ErrPhoneInvalid
	}
	if err := s.codes.Verify(ctx, phone, code); err != nil {
		return nil, err
	}
	id, err := s.repo.GetIdentity(ctx, domain.ProviderPhone, phone)
	if err == nil {
		return s.repo.GetUserByID(ctx, id.UserID)
	}
	if !errors.Is(err, errorx.ErrNotFound) {
		return nil, err
	}
	u := &domain.User{
		ID: uuid.NewString(), Username: "u_" + uuid.NewString()[:8],
		Phone: phone, Nickname: "用户" + lastN(phone, 4), Status: domain.StatusActive,
	}
	newID := &domain.Identity{
		ID: uuid.NewString(), UserID: u.ID,
		Provider: domain.ProviderPhone, ProviderUID: phone,
	}
	if err := s.repo.CreateAccount(ctx, u, newID); err != nil {
		return nil, err
	}
	return u, nil
}

// LoginByOAuth 第三方登录：命中 identity 则登录，否则自动注册（GitHub/Google/Apple 通用）。
func (s *Service) LoginByOAuth(ctx context.Context, provider, providerUID, email, nickname, avatar string) (*domain.User, error) {
	id, err := s.repo.GetIdentity(ctx, provider, providerUID)
	if err == nil {
		return s.repo.GetUserByID(ctx, id.UserID)
	}
	if !errors.Is(err, errorx.ErrNotFound) {
		return nil, err
	}
	if nickname == "" {
		nickname = provider + " 用户"
	}
	u := &domain.User{
		ID: uuid.NewString(), Username: provider + "_" + providerUID,
		Email: email, Nickname: nickname, Avatar: avatar, Status: domain.StatusActive,
	}
	newID := &domain.Identity{
		ID: uuid.NewString(), UserID: u.ID,
		Provider: provider, ProviderUID: providerUID,
	}
	if err := s.repo.CreateAccount(ctx, u, newID); err != nil {
		return nil, err
	}
	return u, nil
}

// GetByID 查询用户（用于 /users/me）。
func (s *Service) GetByID(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetUserByID(ctx, id)
}

func lastN(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[len(s)-n:]
}
