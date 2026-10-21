// Package store 封装 MySQL 数据访问。
package store

import (
	"context"
	"database/sql"
	"errors"
	"time"

	_ "github.com/go-sql-driver/mysql"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("记录不存在")

// Store 数据访问入口。
type Store struct {
	db *sql.DB
}

// Open 建立连接池并做一次连通性检查。
func Open(dsn string) (*Store, error) {
	db, err := sql.Open("mysql", dsn)
	if err != nil {
		return nil, err
	}
	db.SetMaxOpenConns(30)
	db.SetMaxIdleConns(10)
	db.SetConnMaxLifetime(time.Hour)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}
	return &Store{db: db}, nil
}

// Close 关闭连接池。
func (s *Store) Close() error { return s.db.Close() }

// Ping 健康检查。
func (s *Store) Ping(ctx context.Context) error { return s.db.PingContext(ctx) }

// DB 暴露底层连接（供健康检查等使用）。
func (s *Store) DB() *sql.DB { return s.db }

// nullTime 把 sql.NullTime 转成 *time.Time。
func nullTime(t sql.NullTime) *time.Time {
	if !t.Valid {
		return nil
	}
	v := t.Time
	return &v
}

// splitList 把逗号分隔字符串拆成切片并去除空项。
func splitList(s string) []string {
	if s == "" {
		return []string{}
	}
	out := make([]string, 0, 4)
	start := 0
	for i := 0; i <= len(s); i++ {
		if i == len(s) || s[i] == ',' {
			item := s[start:i]
			// trim
			for len(item) > 0 && (item[0] == ' ' || item[0] == '\t') {
				item = item[1:]
			}
			for len(item) > 0 && (item[len(item)-1] == ' ' || item[len(item)-1] == '\t') {
				item = item[:len(item)-1]
			}
			if item != "" {
				out = append(out, item)
			}
			start = i + 1
		}
	}
	return out
}
