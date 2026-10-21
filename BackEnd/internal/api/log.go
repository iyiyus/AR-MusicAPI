package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"ar-musicapi/internal/httpx"
	"ar-musicapi/internal/jwtx"
	"ar-musicapi/internal/store"
)

// handleLogList 分页查询调用日志。
func (s *Server) handleLogList(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	current, size := httpx.NormalizePage(
		httpx.QueryInt(r, "current", 1),
		httpx.QueryInt(r, "size", 20),
	)

	query := store.LogQuery{
		Source:  httpx.QueryStr(r, "source"),
		Keyword: httpx.QueryStr(r, "keyword"),
		Current: current,
		Size:    size,
	}
	if raw := httpx.QueryStr(r, "success"); raw != "" && raw != "all" {
		if b, err := strconv.ParseBool(raw); err == nil {
			query.Success = &b
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	list, total, err := s.store.ListLogs(ctx, query)
	if err != nil {
		httpx.ServerError(w, "日志查询失败")
		return
	}

	httpx.OK(w, map[string]interface{}{
		"records": list,
		"total":   total,
		"current": current,
		"size":    size,
	})
}

// handleLogClear 清理日志：days=0 清空全部，其它值保留最近 N 天。
func (s *Server) handleLogClear(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	days := httpx.QueryInt(r, "days", 0)
	if days < 0 {
		days = 0
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	n, err := s.store.ClearLogs(ctx, days)
	if err != nil {
		httpx.ServerError(w, "日志清理失败")
		return
	}
	httpx.OKMsg(w, "已清理 "+strconv.FormatInt(n, 10)+" 条日志", map[string]interface{}{"count": n})
}
