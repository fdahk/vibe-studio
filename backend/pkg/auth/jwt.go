package auth

import (
	"time"

	"github.com/golang-jwt/jwt/v5"

	"vibe-studio/backend/pkg/errorx"
)

// JWT 用 HS256 签发/校验 token，sub = userID。
type JWT struct {
	secret []byte
	ttl    time.Duration
}

func NewJWT(secret string, ttl time.Duration) *JWT {
	return &JWT{secret: []byte(secret), ttl: ttl}
}

// Issue 为 userID 签发 token。
func (j *JWT) Issue(userID string) (string, error) {
	claims := jwt.RegisteredClaims{
		Subject:   userID,
		IssuedAt:  jwt.NewNumericDate(time.Now()),
		ExpiresAt: jwt.NewNumericDate(time.Now().Add(j.ttl)),
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString(j.secret)
}

// Parse 校验 token 并返回 userID（失败返回 ErrUnauthorized）。
func (j *JWT) Parse(tokenStr string) (string, error) {
	claims := &jwt.RegisteredClaims{}
	t, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errorx.ErrUnauthorized
		}
		return j.secret, nil
	})
	if err != nil || !t.Valid {
		return "", errorx.ErrUnauthorized.WithMsg("token 无效或已过期")
	}
	return claims.Subject, nil
}
