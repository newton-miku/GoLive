// Package repository 提供数据持久化层。
package repository

import (
	"database/sql"
	"strconv"
	"time"

	"golive/internal/domain"
	"golive/internal/log"

	_ "modernc.org/sqlite"
)

// UserRepository 用户数据仓库。
type UserRepository struct {
	db *sql.DB
}

// NewUserRepository 创建用户仓库并初始化数据库。
func NewUserRepository(dbPath string) (*UserRepository, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, err
	}
	if err = db.Ping(); err != nil {
		return nil, err
	}
	repo := &UserRepository{db: db}
	if err := repo.migrate(); err != nil {
		return nil, err
	}
	return repo, nil
}

func (r *UserRepository) migrate() error {
	// 新表结构：id 为自增整数，保留 username 作为唯一标识
	_, _ = r.db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id          INTEGER PRIMARY KEY AUTOINCREMENT,
			username    TEXT UNIQUE NOT NULL,
			password    TEXT NOT NULL,
			room_id     TEXT UNIQUE NOT NULL,
			created_at  INTEGER NOT NULL
		)`)
	_, err := r.db.Exec(`PRAGMA journal_mode=WAL`)
	return err
}

// DB 返回底层数据库连接（供 RoomRepository 共享连接）。
func (r *UserRepository) DB() *sql.DB {
	return r.db
}

// Create 插入用户，返回自增 ID。
func (r *UserRepository) Create(user *domain.User) (int64, error) {
	log.Debugf("[DB] Create user: username=%s room_id=%s\n", user.Username, user.RoomID)
	ts := strconv.FormatInt(user.CreatedAt.Unix(), 10)
	result, err := r.db.Exec(
		`INSERT INTO users (username, password, room_id, created_at) VALUES (?, ?, ?, ?)`,
		user.Username, user.PasswordHash, user.RoomID, ts,
	)
	if err != nil {
		log.Debugf("[DB] Create error: %v\n", err)
		return 0, err
	}
	id, _ := result.LastInsertId()
	return id, nil
}

// GetByUsername 根据用户名查询。
func (r *UserRepository) GetByUsername(username string) (*domain.User, error) {
	row := r.db.QueryRow(
		`SELECT id, username, password, room_id, created_at FROM users WHERE username = ?`,
		username,
	)
	return r.scanUser(row)
}

// GetByID 根据自增 ID 查询。
func (r *UserRepository) GetByID(id int64) (*domain.User, error) {
	row := r.db.QueryRow(
		`SELECT id, username, password, room_id, created_at FROM users WHERE id = ?`,
		id,
	)
	return r.scanUser(row)
}

// ListAll 返回所有用户（用于重启后重建直播间）。
func (r *UserRepository) ListAll() ([]*domain.User, error) {
	rows, err := r.db.Query(`SELECT id, username, password, room_id, created_at FROM users`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var users []*domain.User
	for rows.Next() {
		u, err := r.scanUserRow(rows)
		if err != nil {
			return nil, err
		}
		users = append(users, u)
	}
	return users, rows.Err()
}

func (r *UserRepository) scanUser(row *sql.Row) (*domain.User, error) {
	var u domain.User
	var createdAt int64
	err := row.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.RoomID, &createdAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	u.CreatedAt = time.Unix(createdAt, 0)
	return &u, nil
}

func (r *UserRepository) scanUserRow(rows *sql.Rows) (*domain.User, error) {
	var u domain.User
	var createdAt int64
	if err := rows.Scan(&u.ID, &u.Username, &u.PasswordHash, &u.RoomID, &createdAt); err != nil {
		return nil, err
	}
	u.CreatedAt = time.Unix(createdAt, 0)
	return &u, nil
}

// Close 关闭数据库连接。
func (r *UserRepository) Close() error {
	return r.db.Close()
}

// GetByRoomID 根据房间 ID 查询用户。
func (r *UserRepository) GetByRoomID(roomID string) (*domain.User, error) {
	row := r.db.QueryRow(
		`SELECT id, username, password, room_id, created_at FROM users WHERE room_id = ?`,
		roomID,
	)
	return r.scanUser(row)
}

// GetByRoomIDString 根据房间 ID 查询用户（字符串版本，供外部调用）。
func (r *UserRepository) GetByRoomIDString(roomID string) (*domain.User, error) {
	return r.GetByRoomID(roomID)
}
