// Package auth 提供密码哈希与 JWT 能力。
package auth

import "golang.org/x/crypto/bcrypt"

// HashPassword 用 bcrypt 哈希明文密码（自带 salt）。
func HashPassword(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), bcrypt.DefaultCost)
	return string(b), err
}

// VerifyPassword 校验明文与哈希是否匹配。
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}
