package user

import (
	"vibe-studio/backend/api/openapi"
	userapp "vibe-studio/backend/application/user"
	domain "vibe-studio/backend/domain/user"
)

// svc 由组合根(router.Register)注入，handler 通过它调用应用层。
var svc *userapp.Service

func SetService(s *userapp.Service) { svc = s }

// toModel 领域实体 → OpenAPI 传输模型（不含凭证；时间转 unix 秒）。
func toModel(u *domain.User) openapi.User {
	return openapi.User{
		Id:        u.ID,
		Username:  u.Username,
		Email:     u.Email,
		Phone:     strptr(u.Phone),
		Nickname:  strptr(u.Nickname),
		Avatar:    strptr(u.Avatar),
		Status:    strptr(u.Status),
		CreatedAt: u.CreatedAt.Unix(),
	}
}

func strptr(s string) *string { return &s }
