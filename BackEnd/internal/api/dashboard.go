package api

import (
	"context"
	"net/http"
	"time"

	"ar-musicapi/internal/httpx"
	"ar-musicapi/internal/jwtx"
	"ar-musicapi/internal/store"
)

// handleStats 返回仪表盘汇总指标。
func (s *Server) handleStats(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	stats, err := s.store.GetStats(ctx)
	if err != nil {
		httpx.ServerError(w, "统计数据查询失败")
		return
	}

	defaultSource, err := s.store.GetSetting(ctx, "default_source")
	if err != nil || defaultSource == "" {
		defaultSource = "qqmusic"
	}
	siteName, _ := s.store.GetSetting(ctx, "site_name")
	if siteName == "" {
		siteName = "AR-musicAPI"
	}

	httpx.OK(w, map[string]interface{}{
		"total_calls":     stats.TotalCalls,
		"today_calls":     stats.TodayCalls,
		"success_rate":    stats.SuccessRate,
		"avg_duration_ms": stats.AvgDurationMS,
		"total_cookies":   stats.TotalCookies,
		"active_cookies":  stats.ActiveCookies,
		"cookies":         stats.Cookies,
		"default_source":  defaultSource,
		"site_name":       siteName,
		"sources":         s.registry.Sources(),
		"uptime":          time.Since(s.startAt).Round(time.Second).String(),
	})
}

// handleTrend 返回最近 N 天调用趋势。
func (s *Server) handleTrend(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	days := httpx.QueryInt(r, "days", 7)
	if days <= 0 || days > 90 {
		days = 7
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	points, err := s.store.GetTrend(ctx, days)
	if err != nil {
		httpx.ServerError(w, "趋势数据查询失败")
		return
	}
	httpx.OK(w, points)
}

// handleRecentLogs 返回仪表盘实时日志。
func (s *Server) handleRecentLogs(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	limit := httpx.QueryInt(r, "limit", 10)
	current, size := httpx.NormalizePage(1, limit)

	source := httpx.QueryStr(r, "source")

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	logs, _, err := s.store.ListLogs(ctx, store.LogQuery{
		Source:  source,
		Current: current,
		Size:    size,
	})
	if err != nil {
		httpx.ServerError(w, "日志查询失败")
		return
	}
	httpx.OK(w, logs)
}
