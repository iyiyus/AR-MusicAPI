package store

import (
	"context"
	"database/sql"
)

// GetSetting 读取单个配置，不存在返回空串。
func (s *Store) GetSetting(ctx context.Context, key string) (string, error) {
	var v sql.NullString
	err := s.db.QueryRowContext(ctx, "SELECT key_value FROM settings WHERE key_name = ?", key).Scan(&v)
	if err == sql.ErrNoRows {
		return "", nil
	}
	if err != nil {
		return "", err
	}
	if !v.Valid {
		return "", nil
	}
	return v.String, nil
}

// GetSettings 读取全部配置。
func (s *Store) GetSettings(ctx context.Context) (map[string]string, error) {
	rows, err := s.db.QueryContext(ctx, "SELECT key_name, key_value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]string)
	for rows.Next() {
		var (
			k string
			v sql.NullString
		)
		if err := rows.Scan(&k, &v); err != nil {
			return nil, err
		}
		if v.Valid {
			out[k] = v.String
		} else {
			out[k] = ""
		}
	}
	return out, rows.Err()
}

// SetSetting 写入单个配置（存在则更新）。
func (s *Store) SetSetting(ctx context.Context, key, value string) error {
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO settings (key_name, key_value) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE key_value = VALUES(key_value)`, key, value)
	return err
}

// SetSettings 批量写入配置。
func (s *Store) SetSettings(ctx context.Context, kv map[string]string) error {
	if len(kv) == 0 {
		return nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO settings (key_name, key_value) VALUES (?, ?)
		 ON DUPLICATE KEY UPDATE key_value = VALUES(key_value)`)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for k, v := range kv {
		if _, err := stmt.ExecContext(ctx, k, v); err != nil {
			return err
		}
	}
	return tx.Commit()
}
