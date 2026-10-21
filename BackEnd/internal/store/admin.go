package store

import (
	"context"
	"database/sql"
	"time"
)

const adminColumns = `id, username, password, nickname, avatar, roles, buttons, status,
                      last_login_at, last_login_ip, created_at`

// Admin 管理员账号。
type Admin struct {
	ID          uint       `json:"id"`
	Username    string     `json:"username"`
	Password    string     `json:"-"`
	Nickname    string     `json:"nickname"`
	Avatar      string     `json:"avatar"`
	Roles       []string   `json:"roles"`
	Buttons     []string   `json:"buttons"`
	Status      int        `json:"status"`
	LastLoginAt *time.Time `json:"last_login_at"`
	LastLoginIP string     `json:"last_login_ip"`
	CreatedAt   time.Time  `json:"created_at"`
}

// scanAdmin 统一处理 admins 表的一行扫描。
func scanAdmin(scan func(dest ...interface{}) error) (*Admin, error) {
	var (
		a         Admin
		roles     string
		buttons   string
		lastLogin sql.NullTime
	)
	err := scan(&a.ID, &a.Username, &a.Password, &a.Nickname, &a.Avatar,
		&roles, &buttons, &a.Status, &lastLogin, &a.LastLoginIP, &a.CreatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, err
	}
	a.Roles = splitList(roles)
	a.Buttons = splitList(buttons)
	a.LastLoginAt = nullTime(lastLogin)
	return &a, nil
}

// CountAdmins 统计管理员数量，用于首次启动初始化。
func (s *Store) CountAdmins(ctx context.Context) (int, error) {
	var n int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM admins").Scan(&n)
	return n, err
}

// CreateAdmin 新增管理员。
func (s *Store) CreateAdmin(ctx context.Context, username, passwordHash, nickname, roles, buttons string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO admins (username, password, nickname, roles, buttons) VALUES (?, ?, ?, ?, ?)`,
		username, passwordHash, nickname, roles, buttons,
	)
	return err
}

// GetAdminByUsername 按账号查询管理员。
func (s *Store) GetAdminByUsername(ctx context.Context, username string) (*Admin, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+adminColumns+` FROM admins WHERE username = ? LIMIT 1`, username)
	return scanAdmin(row.Scan)
}

// GetAdminByID 按主键查询管理员。
func (s *Store) GetAdminByID(ctx context.Context, id uint) (*Admin, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+adminColumns+` FROM admins WHERE id = ? LIMIT 1`, id)
	return scanAdmin(row.Scan)
}

// TouchAdminLogin 记录最后登录信息。
func (s *Store) TouchAdminLogin(ctx context.Context, id uint, ip string) error {
	_, err := s.db.ExecContext(ctx,
		`UPDATE admins SET last_login_at = NOW(), last_login_ip = ? WHERE id = ?`, ip, id)
	return err
}

// UpdateAdminPassword 更新密码哈希。
func (s *Store) UpdateAdminPassword(ctx context.Context, id uint, passwordHash string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE admins SET password = ? WHERE id = ?`, passwordHash, id)
	return err
}

// UpdateAdminProfile 更新昵称与头像。
func (s *Store) UpdateAdminProfile(ctx context.Context, id uint, nickname, avatar string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE admins SET nickname = ?, avatar = ? WHERE id = ?`, nickname, avatar, id)
	return err
}
