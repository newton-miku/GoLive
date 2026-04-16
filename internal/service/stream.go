// Package service 提供核心业务逻辑。
package service

import (
	"golive/internal/domain"
	"golive/internal/repository"

	"github.com/google/uuid"
)

// StreamService 直播业务服务。
type StreamService struct {
	repo *repository.RoomRepository
}

// NewStreamService 创建直播服务。
func NewStreamService(repo *repository.RoomRepository) *StreamService {
	return &StreamService{repo: repo}
}

// GetRoom 获取房间。
func (s *StreamService) GetRoom(id string) *domain.Room {
	return s.repo.Get(id)
}

// ListRooms 返回所有房间。
func (s *StreamService) ListRooms() []*domain.Room {
	return s.repo.List()
}

// RecommendRooms 返回推荐排序的房间列表（直播优先，人数降序）。
func (s *StreamService) RecommendRooms() []*domain.Room {
	rooms := s.repo.List()
	for i := 0; i < len(rooms)-1; i++ {
		for j := i + 1; j < len(rooms); j++ {
			ai, aj := rooms[i].IsActive(), rooms[j].IsActive()
			vi, vj := rooms[i].GetViewerCount(), rooms[j].GetViewerCount()
			if (!ai && aj) || (ai == aj && vi < vj) {
				rooms[i], rooms[j] = rooms[j], rooms[i]
			}
		}
	}
	return rooms
}

// CreateRoom 创建房间。
func (s *StreamService) CreateRoom(id, name, password string) *domain.Room {
	room := domain.NewRoom(id, name, password)
	s.repo.Create(room)
	return room
}

// ResetStreamKey 重置指定房间的推流密钥。
func (s *StreamService) ResetStreamKey(roomID string) string {
	newKey := uuid.NewString()[:16]
	s.repo.UpdateStreamKey(roomID, newKey)
	return newKey
}

// OnStreamStart 推流开始回调。
func (s *StreamService) OnStreamStart(room *domain.Room) {
	room.SetStreamActive(true)
}

// OnStreamEnd 推流结束回调。
func (s *StreamService) OnStreamEnd(room *domain.Room) {
	room.SetStreamActive(false)
}

