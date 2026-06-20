package conf

import (
	"os"
	"strings"
)

// Config 是全局配置。对齐 coze-studio 的 conf/ 分层：配置集中、从环境变量加载。
// 右size：用最简单的 env 读取，不引入 viper；够用且零依赖。
type Config struct {
	HTTPAddr string
	MySQL    MySQLConfig
	Redis    RedisConfig
	MinIO    MinIOConfig
	JWT      JWTConfig
	SMS      SMSConfig
	OAuth    OAuthConfig
	Session  SessionConfig
	CORS     CORSConfig
}

type MySQLConfig struct {
	DSN string
}

type RedisConfig struct {
	Addr     string
	Password string
	DB       int
}

type MinIOConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

type JWTConfig struct {
	Secret string
}

// SessionConfig access/refresh 与 refresh cookie 的配置。
type SessionConfig struct {
	AccessTTLMinutes int
	RefreshTTLDays   int
	CookieName       string
	CookieSecure     bool   // 生产 true；本地 http dev false（否则 Secure cookie 不落）
	CookieDomain     string // 默认空=当前主机
}

// CORSConfig 允许携带凭证的跨域来源白名单。
type CORSConfig struct {
	AllowedOrigins []string
}

// SMSConfig 短信验证码相关配置。Provider 预留以后切换真实网关（console=dev 打日志）。
type SMSConfig struct {
	Provider        string
	CodeTTLSeconds  int
	CooldownSeconds int
}

// OAuthConfig 第三方登录配置。未配置 client 时对应 provider 优雅关闭。
type OAuthConfig struct {
	GitHubClientID     string
	GitHubClientSecret string
	GitHubRedirectURL  string // 后端回调，须与 GitHub OAuth App 里填的一致
	FrontendURL        string // 登录成功后跳回的前端地址
}

// Load 从环境变量读取配置，缺省值面向本地 docker-compose。
func Load() *Config {
	return &Config{
		HTTPAddr: env("HTTP_ADDR", ":8888"),
		MySQL: MySQLConfig{
			DSN: env("MYSQL_DSN", "root:root@tcp(127.0.0.1:3306)/vibe_studio?charset=utf8mb4&parseTime=True&loc=Local"),
		},
		Redis: RedisConfig{
			Addr:     env("REDIS_ADDR", "127.0.0.1:6379"),
			Password: env("REDIS_PASSWORD", ""),
			DB:       0,
		},
		MinIO: MinIOConfig{
			Endpoint:  env("MINIO_ENDPOINT", "127.0.0.1:9000"),
			AccessKey: env("MINIO_ACCESS_KEY", "minioadmin"),
			SecretKey: env("MINIO_SECRET_KEY", "minioadmin"),
			Bucket:    env("MINIO_BUCKET", "vibe-studio"),
			UseSSL:    false,
		},
		JWT: JWTConfig{
			Secret: env("JWT_SECRET", "dev-secret-change-me"),
		},
		Session: SessionConfig{
			AccessTTLMinutes: 15,
			RefreshTTLDays:   30,
			CookieName:       env("COOKIE_NAME", "vibe_refresh"),
			CookieSecure:     env("COOKIE_SECURE", "false") == "true",
			CookieDomain:     env("COOKIE_DOMAIN", ""),
		},
		CORS: CORSConfig{
			AllowedOrigins: splitCSV(env("ALLOWED_ORIGINS", "http://localhost:5173")),
		},
		SMS: SMSConfig{
			Provider:        env("SMS_PROVIDER", "console"),
			CodeTTLSeconds:  300,
			CooldownSeconds: 60,
		},
		OAuth: OAuthConfig{
			GitHubClientID:     env("GITHUB_CLIENT_ID", ""),
			GitHubClientSecret: env("GITHUB_CLIENT_SECRET", ""),
			GitHubRedirectURL:  env("OAUTH_GITHUB_REDIRECT_URL", "http://localhost:8888/api/v1/auth/oauth/github/callback"),
			FrontendURL:        env("FRONTEND_URL", "http://localhost:5173"),
		},
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// splitCSV 把逗号分隔串切成去空白的非空项。
func splitCSV(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
