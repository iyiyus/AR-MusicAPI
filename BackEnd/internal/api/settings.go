package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"ar-musicapi/internal/httpx"
	"ar-musicapi/internal/jwtx"
	"golang.org/x/crypto/bcrypt"
)

// defaultSettings 缺省配置。
var defaultSettings = map[string]string{
	"default_source":        "qqmusic",
	"site_name":             "AR-musicAPI",
	"site_icon":             "",
	"log_retention_days":    "30",
	"auto_switch_cookie":    "1",
	"cookie_fail_threshold": "3",
	"allow_public_api":      "1",
}

// 站点图标上传限制。
const maxIconBytes = 2 << 20 // 2MB

// allowedIconExt 允许的图标后缀。
var allowedIconExt = map[string]bool{
	".png":  true,
	".jpg":  true,
	".jpeg": true,
	".webp": true,
	".gif":  true,
	".ico":  true,
	".svg":  true,
}

// handleSettingsGet 返回全部配置与音乐源列表。
func (s *Server) handleSettingsGet(w http.ResponseWriter, r *http.Request, claims *jwtx.Claims) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	kv, err := s.store.GetSettings(ctx)
	if err != nil {
		httpx.ServerError(w, "配置读取失败")
		return
	}

	out := make(map[string]interface{}, len(kv)+3)
	for k, v := range defaultSettings {
		out[k] = v
	}
	for k, v := range kv {
		out[k] = v
	}

	out["sources"] = s.registry.Sources()

	if admin, err := s.store.GetAdminByID(ctx, claims.UserID); err == nil {
		out["account"] = map[string]interface{}{
			"id":           admin.ID,
			"username":     admin.Username,
			"nickname":     admin.Nickname,
			"roles":        admin.Roles,
			"last_login_at": admin.LastLoginAt,
			"last_login_ip": admin.LastLoginIP,
		}
	}

	httpx.OK(w, out)
}

// settingsUpdateRequest 部分更新配置。
type settingsUpdateRequest struct {
	DefaultSource       *string `json:"default_source"`
	SiteName            *string `json:"site_name"`
	SiteIcon            *string `json:"site_icon"`
	LogRetentionDays    *int    `json:"log_retention_days"`
	AutoSwitchCookie    *bool   `json:"auto_switch_cookie"`
	CookieFailThreshold *int    `json:"cookie_fail_threshold"`
	AllowPublicAPI      *bool   `json:"allow_public_api"`
}

// handleSettingsUpdate 更新系统配置。
func (s *Server) handleSettingsUpdate(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	var req settingsUpdateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Fail(w, httpx.CodeBadRequest, "请求参数格式不正确")
		return
	}

	updates := make(map[string]string, 6)

	if req.DefaultSource != nil {
		if _, ok := s.registry.Get(*req.DefaultSource); !ok {
			httpx.Fail(w, httpx.CodeBadRequest, "不支持的音乐源")
			return
		}
		updates["default_source"] = *req.DefaultSource
	}
	if req.SiteName != nil {
		name := strings.TrimSpace(*req.SiteName)
		if name == "" || len([]rune(name)) > 40 {
			httpx.Fail(w, httpx.CodeBadRequest, "站点名称需为 1-40 个字符")
			return
		}
		updates["site_name"] = name
	}
	if req.SiteIcon != nil {
		icon := strings.TrimSpace(*req.SiteIcon)
		if len(icon) > 512 {
			httpx.Fail(w, httpx.CodeBadRequest, "图标地址过长")
			return
		}
		if icon != "" &&
			!strings.HasPrefix(icon, "/") &&
			!strings.HasPrefix(icon, "http://") &&
			!strings.HasPrefix(icon, "https://") {
			httpx.Fail(w, httpx.CodeBadRequest, "图标地址不合法")
			return
		}
		updates["site_icon"] = icon
	}
	if req.LogRetentionDays != nil {
		days := *req.LogRetentionDays
		if days < 1 || days > 365 {
			httpx.Fail(w, httpx.CodeBadRequest, "日志保留天数需在 1-365 之间")
			return
		}
		updates["log_retention_days"] = strconv.Itoa(days)
	}
	if req.AutoSwitchCookie != nil {
		updates["auto_switch_cookie"] = boolToStr(*req.AutoSwitchCookie)
	}
	if req.CookieFailThreshold != nil {
		n := *req.CookieFailThreshold
		if n < 1 || n > 20 {
			httpx.Fail(w, httpx.CodeBadRequest, "失败阈值需在 1-20 之间")
			return
		}
		updates["cookie_fail_threshold"] = strconv.Itoa(n)
	}
	if req.AllowPublicAPI != nil {
		updates["allow_public_api"] = boolToStr(*req.AllowPublicAPI)
	}

	if len(updates) == 0 {
		httpx.Fail(w, httpx.CodeBadRequest, "没有需要更新的内容")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	if err := s.store.SetSettings(ctx, updates); err != nil {
		httpx.ServerError(w, "配置保存失败")
		return
	}
	httpx.OKMsg(w, "配置已保存", nil)
}

// passwordChangeRequest 修改密码请求体。
type passwordChangeRequest struct {
	OldPassword string `json:"oldPassword"`
	NewPassword string `json:"newPassword"`
}

// handlePasswordChange 修改当前账号密码。
func (s *Server) handlePasswordChange(w http.ResponseWriter, r *http.Request, claims *jwtx.Claims) {
	var req passwordChangeRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Fail(w, httpx.CodeBadRequest, "请求参数格式不正确")
		return
	}
	if req.OldPassword == "" || req.NewPassword == "" {
		httpx.Fail(w, httpx.CodeBadRequest, "请填写当前密码与新密码")
		return
	}
	if len([]rune(req.NewPassword)) < 6 {
		httpx.Fail(w, httpx.CodeBadRequest, "新密码至少需要 6 个字符")
		return
	}
	if req.OldPassword == req.NewPassword {
		httpx.Fail(w, httpx.CodeBadRequest, "新密码不能与当前密码相同")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	admin, err := s.store.GetAdminByID(ctx, claims.UserID)
	if err != nil {
		httpx.Fail(w, httpx.CodeUnauthorized, "账号不存在")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.OldPassword)) != nil {
		httpx.Fail(w, httpx.CodeBadRequest, "当前密码不正确")
		return
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		httpx.ServerError(w, "密码加密失败")
		return
	}
	if err := s.store.UpdateAdminPassword(ctx, admin.ID, string(hash)); err != nil {
		httpx.ServerError(w, "密码更新失败")
		return
	}

	httpx.OKMsg(w, "密码修改成功，请重新登录", nil)
}

func boolToStr(b bool) string {
	if b {
		return "1"
	}
	return "0"
}

// handleSiteInfo 返回公开的站点信息（名称、图标），供登录页与前台页面使用。
func (s *Server) handleSiteInfo(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	kv, err := s.store.GetSettings(ctx)
	if err != nil {
		httpx.ServerError(w, "站点信息读取失败")
		return
	}

	name := kv["site_name"]
	if name == "" {
		name = defaultSettings["site_name"]
	}

	httpx.OK(w, map[string]string{
		"site_name": name,
		"site_icon": kv["site_icon"],
	})
}

// handleSettingsUploadIcon 上传站点图标，返回可直接访问的地址。
func (s *Server) handleSettingsUploadIcon(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	r.Body = http.MaxBytesReader(w, r.Body, maxIconBytes+(1<<20))
	if err := r.ParseMultipartForm(maxIconBytes); err != nil {
		httpx.Fail(w, httpx.CodeBadRequest, "图片不能超过 2MB")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		httpx.Fail(w, httpx.CodeBadRequest, "请选择要上传的图片")
		return
	}
	defer func() { _ = file.Close() }()

	ext := strings.ToLower(filepath.Ext(header.Filename))
	if !allowedIconExt[ext] {
		httpx.Fail(w, httpx.CodeBadRequest, "仅支持 png、jpg、jpeg、webp、gif、ico、svg 格式")
		return
	}

	dir := "uploads"
	if err := os.MkdirAll(dir, 0o755); err != nil {
		httpx.ServerError(w, "上传目录创建失败")
		return
	}

	name := fmt.Sprintf("site-icon-%d%s", time.Now().UnixNano(), ext)
	dst, err := os.Create(filepath.Join(dir, name))
	if err != nil {
		httpx.ServerError(w, "图标保存失败")
		return
	}
	defer func() { _ = dst.Close() }()

	if _, err := io.Copy(dst, io.LimitReader(file, maxIconBytes)); err != nil {
		httpx.ServerError(w, "图标保存失败")
		return
	}

	httpx.OK(w, map[string]string{"url": "/uploads/" + name})
}
