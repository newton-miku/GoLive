// Package config 提供全局配置。
package config

import (
	"os"
	"time"
)

// JWT 密钥，从环境变量读取，默认为固定字符串（生产环境应设置 GOPHERJWTSECRET 环境变量）。
var JWTSecret = []byte(func() string {
	if s := os.Getenv("GOLIVE_JWT_SECRET"); s != "" {
		return s
	}
	// 默认密钥，生产务必通过环境变量覆盖
	return "golive-jwt-secret-change-me-in-production-2026"
}())

// TokenExpiry JWT 有效期（30天）。
const TokenExpiry = 30 * 24 * time.Hour

// Issuer JWT 发行者标识。
const Issuer = "golive"
