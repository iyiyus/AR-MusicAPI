package store

import (
	"context"
	"strings"
	"time"
)

// Log API 调用日志。
type Log struct {
	ID            uint      `json:"id"`
	Source        string    `json:"source"`
	SongName      string    `json:"song_name"`
	Singer        string    `json:"singer"`
	SongID        string    `json:"song_id"`
	AlbumImg      string    `json:"album_img"`
	MusicURL      string    `json:"music_url"`
	LyricLRC      string    `json:"lyric_lrc"`
	RequestParams string    `json:"request_params"`
	ResponseCode  int       `json:"response_code"`
	Success       bool      `json:"success"`
	ErrorMsg      string    `json:"error_msg"`
	DurationMS    int       `json:"duration_ms"`
	IP            string    `json:"ip"`
	UserAgent     string    `json:"user_agent"`
	CreatedAt     time.Time `json:"created_at"`
}

// LogQuery 日志列表查询条件。
type LogQuery struct {
	Source  string
	Success *bool
	Keyword string
	Current int
	Size    int
}

func buildLogWhere(q LogQuery) (string, []interface{}) {
	where := make([]string, 0, 3)
	args := make([]interface{}, 0, 4)

	if q.Source != "" && q.Source != "all" {
		where = append(where, "source = ?")
		args = append(args, q.Source)
	}
	if q.Success != nil {
		where = append(where, "success = ?")
		if *q.Success {
			args = append(args, 1)
		} else {
			args = append(args, 0)
		}
	}
	if q.Keyword != "" {
		where = append(where, "(song_name LIKE ? OR ip LIKE ?)")
		like := "%" + q.Keyword + "%"
		args = append(args, like, like)
	}
	if len(where) == 0 {
		return "1=1", args
	}
	return strings.Join(where, " AND "), args
}

// AddLog 写入一条调用日志。
func (s *Store) AddLog(ctx context.Context, l *Log) error {
	success := 0
	if l.Success {
		success = 1
	}
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO api_logs
		   (source, song_name, singer, song_id, album_img, music_url, lyric_lrc,
		    request_params, response_code, success, error_msg, duration_ms, ip, user_agent)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		l.Source, l.SongName, l.Singer, l.SongID, l.AlbumImg, l.MusicURL, l.LyricLRC,
		l.RequestParams, l.ResponseCode, success, l.ErrorMsg, l.DurationMS, l.IP, l.UserAgent)
	if err != nil {
		return err
	}
	if id, err := res.LastInsertId(); err == nil {
		l.ID = uint(id)
	}
	return nil
}

// ListLogs 分页查询日志。
func (s *Store) ListLogs(ctx context.Context, q LogQuery) ([]Log, int, error) {
	cond, args := buildLogWhere(q)

	var total int
	if err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM api_logs WHERE "+cond, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	if total == 0 {
		return []Log{}, 0, nil
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
		`SELECT id, source, song_name, singer, song_id, album_img, music_url, lyric_lrc,
		        request_params, response_code, success, error_msg, duration_ms, ip, user_agent, created_at
		   FROM api_logs WHERE `+cond+`
		  ORDER BY id DESC
		  LIMIT ?, ?`, append(args, offset, size)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	list := make([]Log, 0, size)
	for rows.Next() {
		var (
			l       Log
			success int
		)
		if err := rows.Scan(&l.ID, &l.Source, &l.SongName, &l.Singer, &l.SongID, &l.AlbumImg,
			&l.MusicURL, &l.LyricLRC, &l.RequestParams, &l.ResponseCode, &success,
			&l.ErrorMsg, &l.DurationMS, &l.IP, &l.UserAgent, &l.CreatedAt); err != nil {
			return nil, 0, err
		}
		l.Success = success == 1
		list = append(list, l)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	return list, total, nil
}

// ClearLogs 清理日志：keepDays<=0 表示清空全部。
func (s *Store) ClearLogs(ctx context.Context, keepDays int) (int64, error) {
	if keepDays <= 0 {
		res, err := s.db.ExecContext(ctx, "DELETE FROM api_logs")
		if err != nil {
			return 0, err
		}
		return res.RowsAffected()
	}
	res, err := s.db.ExecContext(ctx,
		"DELETE FROM api_logs WHERE created_at < DATE_SUB(NOW(), INTERVAL ? DAY)", keepDays)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

// Stats 仪表盘汇总数据。
type Stats struct {
	TotalCalls    int64          `json:"total_calls"`
	TodayCalls    int64          `json:"today_calls"`
	SuccessRate   float64        `json:"success_rate"`
	AvgDurationMS int            `json:"avg_duration_ms"`
	TotalCookies  int            `json:"total_cookies"`
	ActiveCookies int            `json:"active_cookies"`
	Cookies       map[string]int `json:"cookies"`
}

// GetStats 汇总仪表盘指标。
func (s *Store) GetStats(ctx context.Context) (*Stats, error) {
	st := &Stats{Cookies: map[string]int{}}

	var (
		rate   float64
		avgDur float64
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT COUNT(*),
		        COALESCE(SUM(DATE(created_at) = CURDATE()), 0),
		        COALESCE(AVG(CASE WHEN success = 1 THEN 100 ELSE 0 END), 0),
		        COALESCE(AVG(duration_ms), 0)
		   FROM api_logs`).Scan(&st.TotalCalls, &st.TodayCalls, &rate, &avgDur)
	if err != nil {
		return nil, err
	}
	st.SuccessRate = float64(int(rate*100+0.5)) / 100
	st.AvgDurationMS = int(avgDur + 0.5)

	if st.Cookies, err = s.CountCookiesBySource(ctx); err != nil {
		return nil, err
	}
	if st.TotalCookies, st.ActiveCookies, err = s.CountCookies(ctx); err != nil {
		return nil, err
	}
	return st, nil
}

// TrendPoint 单日趋势。
type TrendPoint struct {
	Date    string `json:"date"`
	FullDate string `json:"full_date"`
	Count   int    `json:"count"`
	Success int    `json:"success"`
	Failed  int    `json:"failed"`
}

// GetTrend 返回最近 days 天（含今天）的调用趋势。
func (s *Store) GetTrend(ctx context.Context, days int) ([]TrendPoint, error) {
	if days <= 0 {
		days = 7
	}
	if days > 90 {
		days = 90
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT DATE_FORMAT(created_at, '%Y-%m-%d') AS d,
		        COUNT(*) AS total,
		        COALESCE(CAST(SUM(success = 1) AS SIGNED), 0) AS ok
		   FROM api_logs
		  WHERE created_at >= DATE_SUB(CURDATE(), INTERVAL ? DAY)
		  GROUP BY d`, days-1)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	type agg struct{ total, ok int }
	byDate := make(map[string]agg)
	for rows.Next() {
		var (
			d     string
			total int
			ok    int
		)
		if err := rows.Scan(&d, &total, &ok); err != nil {
			return nil, err
		}
		byDate[d] = agg{total: total, ok: ok}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	out := make([]TrendPoint, 0, days)
	now := time.Now()
	for i := days - 1; i >= 0; i-- {
		d := now.AddDate(0, 0, -i)
		key := d.Format("2006-01-02")
		item := byDate[key]
		out = append(out, TrendPoint{
			Date:     d.Format("01/02"),
			FullDate: key,
			Count:    item.total,
			Success:  item.ok,
			Failed:   item.total - item.ok,
		})
	}
	return out, nil
}
