package store

import (
	"context"
	"database/sql"
	"strings"
	"time"
)

// Cookie Cookie 池记录。
type Cookie struct {
	ID        uint       `json:"id"`
	Source    string     `json:"source"`
	Cookie    string     `json:"cookie"`
	Remark    string     `json:"remark"`
	Status    int        `json:"status"`
	FailCount int        `json:"fail_count"`
	CallCount int        `json:"call_count"`
	LastUsed  *time.Time `json:"last_used"`
	CreatedAt time.Time  `json:"created_at"`
	UpdatedAt time.Time  `json:"updated_at"`
}

// CookieQuery Cookie 列表查询条件。
type CookieQuery struct {
	Source  string
	Keyword string
	Status  *int
	Current int
	Size    int
}

func buildCookieWhere(q CookieQuery) (string, []interface{}) {
	where := make([]string, 0, 3)
	args := make([]interface{}, 0, 4)

	if q.Source != "" {
		where = append(where, "source = ?")
		args = append(args, q.Source)
	}
	if q.Status != nil {
		where = append(where, "status = ?")
		args = append(args, *q.Status)
	}
	if q.Keyword != "" {
		where = append(where, "(remark LIKE ? OR cookie LIKE ?)")
		like := "%" + q.Keyword + "%"
		args = append(args, like, like)
	}
	if len(where) == 0 {
		return "1=1", args
	}
	return strings.Join(where, " AND "), args
}

// ListCookies 分页查询 Cookie。
func (s *Store) ListCookies(ctx context.Context, q CookieQuery) ([]Cookie, int, error) {
	cond, args := buildCookieWhere(q)

	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM cookies WHERE "+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []Cookie{}, 0, nil
	}

	current, size := q.Current, q.Size
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 20
	}
	offset := (current - 1) * size

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, source, cookie, remark, status, fail_count, call_count, last_used, created_at, updated_at
		   FROM cookies WHERE `+cond+`
		  ORDER BY source ASC, status DESC, fail_count ASC, id ASC
		  LIMIT ?, ?`, append(args, offset, size)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]Cookie, 0, size)
	for rows.Next() {
		var (
			c        Cookie
			lastUsed sql.NullTime
		)
		if err := rows.Scan(&c.ID, &c.Source, &c.Cookie, &c.Remark, &c.Status,
			&c.FailCount, &c.CallCount, &lastUsed, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, 0, err
		}
		c.LastUsed = nullTime(lastUsed)
		list = append(list, c)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// AddCookies 批量新增 Cookie，返回成功条数。
func (s *Store) AddCookies(ctx context.Context, items []Cookie) (int, error) {
	if len(items) == 0 {
		return 0, nil
	}
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return 0, err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx,
		`INSERT INTO cookies (source, cookie, remark, status) VALUES (?, ?, ?, ?)`)
	if err != nil {
		return 0, err
	}
	defer stmt.Close()

	count := 0
	for _, item := range items {
		cookie := strings.TrimSpace(item.Cookie)
		if cookie == "" {
			continue
		}
		status := item.Status
		if status != 0 {
			status = 1
		}
		if _, err := stmt.ExecContext(ctx, item.Source, cookie, item.Remark, status); err != nil {
			return count, err
		}
		count++
	}
	if err := tx.Commit(); err != nil {
		return count, err
	}
	return count, nil
}

// UpdateCookie 修改单条 Cookie。
func (s *Store) UpdateCookie(ctx context.Context, c Cookie) error {
	res, err := s.db.ExecContext(ctx,
		`UPDATE cookies SET source = ?, cookie = ?, remark = ?, status = ? WHERE id = ?`,
		c.Source, strings.TrimSpace(c.Cookie), c.Remark, c.Status, c.ID)
	if err != nil {
		return err
	}
	n, _ := res.RowsAffected()
	if n == 0 {
		return ErrNotFound
	}
	return nil
}

func placeholders(n int) string {
	if n <= 0 {
		return ""
	}
	var b strings.Builder
	for i := 0; i < n; i++ {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteByte('?')
	}
	return b.String()
}

func toArgs(ids []uint) []interface{} {
	args := make([]interface{}, len(ids))
	for i, id := range ids {
		args[i] = id
	}
	return args
}

// DeleteCookies 批量删除。
func (s *Store) DeleteCookies(ctx context.Context, ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res, err := s.db.ExecContext(ctx,
		"DELETE FROM cookies WHERE id IN ("+placeholders(len(ids))+")", toArgs(ids)...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ToggleCookies 批量切换启用状态。
func (s *Store) ToggleCookies(ctx context.Context, ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res, err := s.db.ExecContext(ctx,
		"UPDATE cookies SET status = 1 - status WHERE id IN ("+placeholders(len(ids))+")", toArgs(ids)...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// ResetCookies 批量重置失败计数并启用。
func (s *Store) ResetCookies(ctx context.Context, ids []uint) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}
	res, err := s.db.ExecContext(ctx,
		"UPDATE cookies SET fail_count = 0, status = 1 WHERE id IN ("+placeholders(len(ids))+")", toArgs(ids)...)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// PickCookie 按「最少使用 + 失败次数最少」挑选一个可用 Cookie。
// 若全部达到失败阈值，会重置该源下的失败计数后重试。
func (s *Store) PickCookie(ctx context.Context, source string, failThreshold int) (string, error) {
	if failThreshold < 1 {
		failThreshold = 3
	}

	var cookie string
	err := s.db.QueryRowContext(ctx,
		`SELECT cookie FROM cookies
		  WHERE source = ? AND status = 1 AND fail_count < ?
		  ORDER BY last_used ASC, fail_count ASC
		  LIMIT 1`, source, failThreshold).Scan(&cookie)

	if err == sql.ErrNoRows {
		if _, rerr := s.db.ExecContext(ctx,
			`UPDATE cookies SET fail_count = 0 WHERE source = ?`, source); rerr != nil {
			return "", rerr
		}
		err = s.db.QueryRowContext(ctx,
			`SELECT cookie FROM cookies WHERE source = ? AND status = 1
			  ORDER BY fail_count ASC, last_used ASC LIMIT 1`, source).Scan(&cookie)
	}
	if err != nil {
		if err == sql.ErrNoRows {
			return "", ErrNotFound
		}
		return "", err
	}

	_, _ = s.db.ExecContext(ctx,
		`UPDATE cookies SET last_used = NOW(), call_count = call_count + 1
		  WHERE source = ? AND cookie = ?`, source, cookie)

	return cookie, nil
}

// ReportCookie 上报一次调用结果，用于自动降权。
func (s *Store) ReportCookie(ctx context.Context, source, cookie string, ok bool) error {
	if cookie == "" {
		return nil
	}
	if ok {
		_, err := s.db.ExecContext(ctx,
			`UPDATE cookies SET fail_count = 0, last_used = NOW() WHERE source = ? AND cookie = ?`,
			source, cookie)
		return err
	}
	_, err := s.db.ExecContext(ctx,
		`UPDATE cookies SET fail_count = fail_count + 1 WHERE source = ? AND cookie = ?`,
		source, cookie)
	return err
}

// CountCookiesBySource 统计各音乐源可用 Cookie 数量。
func (s *Store) CountCookiesBySource(ctx context.Context) (map[string]int, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT source, COUNT(*) FROM cookies WHERE status = 1 GROUP BY source`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	out := make(map[string]int)
	for rows.Next() {
		var (
			source string
			n      int
		)
		if err := rows.Scan(&source, &n); err != nil {
			return nil, err
		}
		out[source] = n
	}
	return out, rows.Err()
}

// CountCookies 统计 Cookie 总数与启用数量。
func (s *Store) CountCookies(ctx context.Context) (total int, active int, err error) {
	err = s.db.QueryRowContext(ctx,
		`SELECT COUNT(*), COALESCE(CAST(SUM(status = 1) AS SIGNED), 0) FROM cookies`).Scan(&total, &active)
	return
}
