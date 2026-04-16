// Package domain 定义核心业务实体和接口。
package domain

import "time"

// User 用户实体。
type User struct {
	ID           int64     // 自增 ID
	Username     string    // 用户名（唯一）
	PasswordHash string    // bcrypt 哈希后的密码
	RoomID       string    // 用户专属房间 ID
	CreatedAt    time.Time // 注册时间
}

// NewUser 创建新用户。
func NewUser(id int64, username, passwordHash, roomID string) *User {
	return &User{
		ID:           id,
		Username:     username,
		PasswordHash: passwordHash,
		RoomID:       roomID,
		CreatedAt:    time.Now(),
	}
}
