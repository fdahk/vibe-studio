package conf

import "os"

// Config 是全局配置。对齐 coze-studio 的 conf/ 分层：配置集中、从环境变量加载。
// 右size：用最简单的 env 读取，不引入 viper；够用且零依赖。
type Config struct {
	HTTPAddr string
	MySQL    MySQLConfig
	Redis    RedisConfig
	MinIO    MinIOConfig
	JWT      JWTConfig
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
	Secret   string
	TTLHours int
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
			Secret:   env("JWT_SECRET", "dev-secret-change-me"),
			TTLHours: 168,
		},
	}
}

func env(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
