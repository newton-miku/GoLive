// Package repository 提供房间数据的持久化存储实现。
package repository

import (
	"database/sql"
	"strconv"
	"sync"
	"time"

	"golive/internal/domain"

	_ "modernc.org/sqlite"
)

// RoomRepository 房间仓库（持久化 + 内存缓存）。
type RoomRepository struct {
	db    *sql.DB
	mu    sync.RWMutex
	rooms map[string]*domain.Room
}

// NewRoomRepository 创建房间仓库。
func NewRoomRepository(db *sql.DB) *RoomRepository {
	repo := &RoomRepository{db: db, rooms: make(map[string]*domain.Room)}
	repo.migrate()
	return repo
}

func (r *RoomRepository) migrate() {
	// 仅在表不存在时创建
	r.db.Exec(`
		CREATE TABLE IF NOT EXISTS rooms (
			room_id    TEXT PRIMARY KEY,
			name       TEXT NOT NULL,
			stream_key TEXT NOT NULL,
			created_at INTEGER NOT NULL
		)`)
	// 已有表可能缺少 created_at 列，忽略错误
	r.db.Exec(`ALTER TABLE rooms ADD COLUMN created_at INTEGER NOT NULL DEFAULT 0`)
}

// LoadAll 从数据库加载所有房间到内存（启动时调用）。
func (r *RoomRepository) LoadAll() error {
	rows, err := r.db.Query(`SELECT room_id, name, stream_key FROM rooms`)
	if err != nil {
		return err
	}
	defer rows.Close()

	r.mu.Lock()
	defer r.mu.Unlock()

	for rows.Next() {
		var roomID, name, streamKey string
		if err := rows.Scan(&roomID, &name, &streamKey); err != nil {
			return err
		}
		r.rooms[roomID] = domain.NewRoom(roomID, name, streamKey)
	}
	return rows.Err()
}

// Create 添加房间并持久化。
func (r *RoomRepository) Create(room *domain.Room) {
	r.mu.Lock()
	r.rooms[room.ID] = room
	r.mu.Unlock()
	_, err := r.db.Exec(`INSERT INTO rooms (room_id, name, stream_key, created_at) VALUES (?, ?, ?, ?)`,
		room.ID, room.Name, room.Password, strconv.FormatInt(time.Now().Unix(), 10))
	_ = err // 忽略写入错误（表不存在等）
}

// Get 根据 ID 获取房间。
func (r *RoomRepository) Get(id string) *domain.Room {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.rooms[id]
}

// List 返回所有房间。
func (r *RoomRepository) List() []*domain.Room {
	r.mu.RLock()
	defer r.mu.RUnlock()
	rooms := make([]*domain.Room, 0, len(r.rooms))
	for _, room := range r.rooms {
		rooms = append(rooms, room)
	}
	return rooms
}

// Exists 检查房间是否存在。
func (r *RoomRepository) Exists(id string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()
	_, ok := r.rooms[id]
	return ok
}

// UpdateStreamKey 更新房间推流密钥并持久化。
func (r *RoomRepository) UpdateStreamKey(roomID, newKey string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if room, ok := r.rooms[roomID]; ok {
		room.SetPassword(newKey)
	}
	r.db.Exec(`UPDATE rooms SET stream_key = ? WHERE room_id = ?`, newKey, roomID)
}

// Delete 删除房间。
func (r *RoomRepository) Delete(id string) {
	r.mu.Lock()
	delete(r.rooms, id)
	r.mu.Unlock()
	r.db.Exec(`DELETE FROM rooms WHERE room_id = ?`, id)
}
