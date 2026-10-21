// Package api 提供 HTTP 接口。
package api

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"runtime/debug"
	"strconv"
	"strings"
	"time"

	"ar-musicapi/internal/config"
	"ar-musicapi/internal/httpx"
	"ar-musicapi/internal/jwtx"
	"ar-musicapi/internal/music"
	"ar-musicapi/internal/store"
)

// Server 聚合依赖并提供路由。
type Server struct {
	cfg     *config.Config
	store   *store.Store
	registry *music.Registry
	jwt     *jwtx.Manager
	startAt time.Time
}

// New 创建 Server。
func New(cfg *config.Config, st *store.Store, reg *music.Registry) *Server {
	return &Server{
		cfg:      cfg,
		store:    st,
		registry: reg,
		jwt:      jwtx.NewManager(cfg.JWT.Secret),
		startAt:  time.Now(),
	}
}

// Handler 构建完整路由（含中间件）。
func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()

	// ---- 认证 ----
	mux.HandleFunc("POST /api/auth/login", s.handleLogin)
	mux.HandleFunc("POST /api/auth/refresh", s.handleRefresh)
	mux.HandleFunc("POST /api/auth/logout", s.handleLogout)
	mux.HandleFunc("GET /api/user/info", s.auth(s.handleUserInfo))
	mux.HandleFunc("PUT /api/user/profile", s.auth(s.handleUpdateProfile))

	// ---- 仪表盘 ----
	mux.HandleFunc("GET /api/dashboard/stats", s.auth(s.handleStats))
	mux.HandleFunc("GET /api/dashboard/trend", s.auth(s.handleTrend))
	mux.HandleFunc("GET /api/dashboard/recent-logs", s.auth(s.handleRecentLogs))

	// ---- Cookie 池 ----
	mux.HandleFunc("GET /api/cookie/list", s.auth(s.handleCookieList))
	mux.HandleFunc("POST /api/cookie/add", s.auth(s.handleCookieAdd))
	mux.HandleFunc("PUT /api/cookie/update", s.auth(s.handleCookieUpdate))
	mux.HandleFunc("DELETE /api/cookie/delete", s.auth(s.handleCookieDelete))
	mux.HandleFunc("POST /api/cookie/toggle", s.auth(s.handleCookieToggle))
	mux.HandleFunc("POST /api/cookie/reset", s.auth(s.handleCookieReset))

	// ---- 调用日志 ----
	mux.HandleFunc("GET /api/log/list", s.auth(s.handleLogList))
	mux.HandleFunc("DELETE /api/log/clear", s.auth(s.handleLogClear))

	// ---- 系统设置 ----
	mux.HandleFunc("GET /api/settings", s.auth(s.handleSettingsGet))
	mux.HandleFunc("PUT /api/settings", s.auth(s.handleSettingsUpdate))
	mux.HandleFunc("PUT /api/settings/password", s.auth(s.handlePasswordChange))
	mux.HandleFunc("POST /api/settings/icon", s.auth(s.handleSettingsUploadIcon))

	// ---- 站点公开信息 ----
	mux.HandleFunc("GET /api/site/info", s.handleSiteInfo)

	// ---- 对外点歌接口 ----
	mux.HandleFunc("GET /api/music/sources", s.handleMusicSources)
	mux.HandleFunc("GET /api/music/search", s.handleMusicSearch)
	mux.HandleFunc("GET /api/health", s.handleHealth)

	// ---- 上传的静态资源（站点图标等） ----
	mux.Handle("/uploads/", http.StripPrefix("/uploads/", http.FileServer(http.Dir("uploads"))))

	// ---- 静态资源（前端打包产物，可选） ----
	if dir := s.staticDir(); dir != "" {
		mux.Handle("/", spaHandler(dir))
		log.Printf("[info] 静态资源目录: %s", dir)
	}

	return s.withMiddleware(mux)
}

// staticDir 返回存在的前端构建目录。
func (s *Server) staticDir() string {
	candidates := []string{
		filepath.Join("..", "FrontEnd", "dist"),
		"public",
		"dist",
	}
	for _, dir := range candidates {
		if info, err := os.Stat(filepath.Join(dir, "index.html")); err == nil && !info.IsDir() {
			return dir
		}
	}
	return ""
}

// withMiddleware 组合 panic 恢复与 CORS。
func (s *Server) withMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				log.Printf("[panic] %s %s: %v\n%s", r.Method, r.URL.Path, rec, debug.Stack())
				httpx.ServerError(w, "服务器内部错误")
			}
		}()

		origin := r.Header.Get("Origin")
		if origin != "" {
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Credentials", "true")
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Requested-With")
			w.Header().Set("Access-Control-Max-Age", "86400")
		}
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// authHandler 已鉴权的处理函数。
type authHandler func(w http.ResponseWriter, r *http.Request, claims *jwtx.Claims)

// auth 校验 Authorization 头中的令牌。
func (s *Server) auth(next authHandler) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		token := r.Header.Get("Authorization")
		if strings.TrimSpace(token) == "" {
			httpx.Fail(w, httpx.CodeUnauthorized, "未登录或登录已过期")
			return
		}

		claims, err := s.jwt.Parse(token)
		if err != nil {
			if errors.Is(err, jwtx.ErrExpiredToken) {
				httpx.Fail(w, httpx.CodeUnauthorized, "登录已过期，请重新登录")
				return
			}
			httpx.Fail(w, httpx.CodeUnauthorized, "无效的登录凭证")
			return
		}
		if claims.TokenTyp != jwtx.TypeAccess {
			httpx.Fail(w, httpx.CodeUnauthorized, "令牌类型不正确")
			return
		}

		next(w, r, claims)
	}
}

// handleHealth 健康检查。
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
	defer cancel()

	dbOK := true
	if err := s.store.Ping(ctx); err != nil {
		dbOK = false
	}

	httpx.OK(w, map[string]interface{}{
		"name":        "AR-musicAPI",
		"status":      map[bool]string{true: "ok", false: "degraded"}[dbOK],
		"database":    dbOK,
		"uptime":      time.Since(s.startAt).Round(time.Second).String(),
		"server_time": time.Now().Format("2006-01-02 15:04:05"),
	})
}

// spaHandler 提供前端静态资源，未命中时回退到 index.html。
func spaHandler(dir string) http.Handler {
	fs := http.FileServer(http.Dir(dir))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		clean := filepath.Clean(strings.TrimPrefix(r.URL.Path, "/"))
		full := filepath.Join(dir, clean)
		if info, err := os.Stat(full); err == nil && !info.IsDir() {
			fs.ServeHTTP(w, r)
			return
		}
		http.ServeFile(w, r, filepath.Join(dir, "index.html"))
	})
}

// parseInt 解析字符串为整数，失败返回默认值。
func parseInt(s string, def int) int {
	if s == "" {
		return def
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return def
	}
	return n
}
