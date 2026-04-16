// Package service 提供核心业务逻辑。
package service

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"golive/internal/config"
	"golive/internal/domain"
	"golive/internal/log"
	"golive/internal/repository"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
	"math/rand"
	"strconv"
)

var (
	ErrUserNotFound     = errors.New("用户不存在")
	ErrUserExists       = errors.New("用户名已存在")
	ErrInvalidPassword  = errors.New("密码错误")
	ErrInvalidUsername  = errors.New("用户名格式错误（3-20位字母数字下划线）")
	ErrInvalidPassword2 = errors.New("密码格式错误（6-32位）")
)

// UserService 用户业务服务。
type UserService struct {
	userRepo *repository.UserRepository
	roomSvc  *StreamService
}

// NewUserService 创建用户服务。
func NewUserService(userRepo *repository.UserRepository, roomSvc *StreamService) *UserService {
	return &UserService{userRepo: userRepo, roomSvc: roomSvc}
}

// Register 注册新用户，自动分配直播间。
func (s *UserService) Register(username, password string) (*domain.User, error) {
	username = strings.TrimSpace(username)
	password = strings.TrimSpace(password)

	if !isValidUsername(username) {
		return nil, ErrInvalidUsername
	}
	if !isValidPassword(password) {
		return nil, ErrInvalidPassword2
	}

	// 检查用户名是否已存在
	existing, err := s.userRepo.GetByUsername(username)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrUserExists
	}

	// 生成 4 位数字房间 ID，避免冲突最多重试 10 次
	var roomID string
	for i := 0; i < 10; i++ {
		roomID = generateRoomID()
		if s.roomSvc.GetRoom(roomID) == nil {
			break
		}
	}
	if s.roomSvc.GetRoom(roomID) != nil {
		return nil, errors.New("房间 ID 冲突，请重试")
	}

	streamKey := generateStreamKey()

	// 创建直播间
	s.roomSvc.CreateRoom(roomID, username+"的直播间", streamKey)

	// 哈希密码
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Debugf("[注册] bcrypt 失败: %v\n", err)
		return nil, err
	}

	// 创建用户记录（自增 ID 由数据库分配）
	user := domain.NewUser(0, username, string(hash), roomID)
	id, err := s.userRepo.Create(user)
	if err != nil {
		log.Debugf("[注册] 数据库插入失败: %v\n", err)
		return nil, err
	}
	user.ID = id
	log.Debugf("[注册] 创建用户: id=%d username=%s roomID=%s\n", user.ID, user.Username, user.RoomID)
	return user, nil
}

// Login 验证用户名和密码。
func (s *UserService) Login(username, password string) (*domain.User, error) {
	user, err := s.userRepo.GetByUsername(strings.TrimSpace(username))
	if err != nil {
		return nil, err
	}
	if user == nil {
		return nil, ErrUserNotFound
	}
	if err = bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, ErrInvalidPassword
	}
	return user, nil
}

// GetUserByID 根据 ID 获取用户。
func (s *UserService) GetUserByID(id int64) (*domain.User, error) {
	return s.userRepo.GetByID(id)
}

// ResetStreamKey 重置指定用户的推流密钥。
func (s *UserService) ResetStreamKey(userID int64) (string, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrUserNotFound
	}
	return s.roomSvc.ResetStreamKey(user.RoomID), nil
}

// GetStreamKey 获取指定用户的推流密钥。
func (s *UserService) GetStreamKey(userID int64) (string, error) {
	user, err := s.userRepo.GetByID(userID)
	if err != nil {
		return "", err
	}
	if user == nil {
		return "", ErrUserNotFound
	}
	room := s.roomSvc.GetRoom(user.RoomID)
	if room == nil {
		return "", nil
	}
	return room.Password, nil
}

// isValidUsername 检查用户名格式（3-20位字母数字下划线）。
func isValidUsername(u string) bool {
	if len(u) < 3 || len(u) > 20 {
		return false
	}
	for _, c := range u {
		if !(c >= 'a' && c <= 'z') && !(c >= 'A' && c <= 'Z') && !(c >= '0' && c <= '9') && c != '_' {
			return false
		}
	}
	return true
}

// isValidPassword 检查密码格式（6-32位）。
func isValidPassword(p string) bool {
	return len(p) >= 6 && len(p) <= 32
}

// generateStreamKey 生成随机推流密钥（仅字母数字，更稳定）。
func generateStreamKey() string {
	const chars = "abcdef0123456789"
	b := make([]byte, 16)
	for i := range b {
		b[i] = chars[rand.Intn(len(chars))]
	}
	return string(b)
}

// generateRoomID 生成 4 位数字房间 ID（1000-9999）。
func generateRoomID() string {
	return strconv.FormatInt(int64(rand.Intn(9000)+1000), 10)
}

// GenerateToken 为用户生成 JWT（sub 字段为用户 ID）。
func GenerateToken(user *domain.User) (string, error) {
	claims := jwt.MapClaims{
		"sub":      fmt.Sprintf("%d", user.ID),
		"username": user.Username,
		"roomId":   user.RoomID,
		"exp":      time.Now().Add(config.TokenExpiry).Unix(),
		"iat":      time.Now().Unix(),
		"iss":      config.Issuer,
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(config.JWTSecret)
}

// ValidateToken 验证 JWT 并返回用户 ID（int64）。
func ValidateToken(tokenStr string) (int64, error) {
	token, err := jwt.Parse(tokenStr, func(token *jwt.Token) (any, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return config.JWTSecret, nil
	})
	if err != nil {
		return 0, err
	}
	if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
		if sub, ok := claims["sub"].(string); ok {
			id, err := strconv.ParseInt(sub, 10, 64)
			if err != nil {
				return 0, err
			}
			return id, nil
		}
	}
	return 0, errors.New("invalid token")
}
