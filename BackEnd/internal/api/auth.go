package api

import (
	"context"
	"net/http"
	"time"

	"ar-musicapi/internal/httpx"
	"ar-musicapi/internal/jwtx"
	"ar-musicapi/internal/store"
	"golang.org/x/crypto/bcrypt"
)

// loginRequest 登录请求体。
type loginRequest struct {
	UserName string `json:"userName"`
	Password string `json:"password"`
}

// loginResponse 登录响应体。
type loginResponse struct {
	Token        string `json:"token"`
	RefreshToken string `json:"refreshToken"`
}

// handleLogin 管理员登录。
func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Fail(w, httpx.CodeBadRequest, "请求参数格式不正确")
		return
	}
	if req.UserName == "" || req.Password == "" {
		httpx.Fail(w, httpx.CodeBadRequest, "请输入账号和密码")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	admin, err := s.store.GetAdminByUsername(ctx, req.UserName)
	if err != nil {
		if err == store.ErrNotFound {
			httpx.Fail(w, httpx.CodeUnauthorized, "账号或密码错误")
			return
		}
		httpx.ServerError(w, "查询账号失败")
		return
	}
	if admin.Status != 1 {
		httpx.Fail(w, httpx.CodeForbidden, "该账号已被禁用")
		return
	}
	if bcrypt.CompareHashAndPassword([]byte(admin.Password), []byte(req.Password)) != nil {
		httpx.Fail(w, httpx.CodeUnauthorized, "账号或密码错误")
		return
	}

	resp, err := s.issueTokens(admin)
	if err != nil {
		httpx.ServerError(w, "生成登录凭证失败")
		return
	}

	_ = s.store.TouchAdminLogin(ctx, admin.ID, httpx.ClientIP(r))

	httpx.OKMsg(w, "登录成功", resp)
}

// issueTokens 签发访问令牌与刷新令牌。
func (s *Server) issueTokens(admin *store.Admin) (*loginResponse, error) {
	access, err := s.jwt.Issue(
		admin.ID, admin.Username, admin.Roles,
		time.Duration(s.cfg.JWT.ExpireHours)*time.Hour, jwtx.TypeAccess)
	if err != nil {
		return nil, err
	}
	refresh, err := s.jwt.Issue(
		admin.ID, admin.Username, admin.Roles,
		time.Duration(s.cfg.JWT.RefreshHours)*time.Hour, jwtx.TypeRefresh)
	if err != nil {
		return nil, err
	}
	return &loginResponse{Token: access, RefreshToken: refresh}, nil
}

// refreshRequest 刷新令牌请求体。
type refreshRequest struct {
	RefreshToken string `json:"refreshToken"`
}

// handleRefresh 用刷新令牌换取新的访问令牌。
func (s *Server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	var req refreshRequest
	if err := httpx.DecodeJSON(r, &req); err != nil || req.RefreshToken == "" {
		httpx.Fail(w, httpx.CodeBadRequest, "缺少 refreshToken")
		return
	}

	claims, err := s.jwt.Parse(req.RefreshToken)
	if err != nil || claims.TokenTyp != jwtx.TypeRefresh {
		httpx.Fail(w, httpx.CodeUnauthorized, "刷新令牌无效或已过期")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	admin, err := s.store.GetAdminByID(ctx, claims.UserID)
	if err != nil || admin.Status != 1 {
		httpx.Fail(w, httpx.CodeUnauthorized, "账号不可用")
		return
	}

	resp, err := s.issueTokens(admin)
	if err != nil {
		httpx.ServerError(w, "生成登录凭证失败")
		return
	}
	httpx.OK(w, resp)
}

// handleLogout 登出（无状态令牌，仅作语义占位）。
func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	httpx.OKMsg(w, "已安全退出", nil)
}

// handleUserInfo 返回当前登录管理员信息，供前端路由守卫使用。
func (s *Server) handleUserInfo(w http.ResponseWriter, r *http.Request, claims *jwtx.Claims) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	admin, err := s.store.GetAdminByID(ctx, claims.UserID)
	if err != nil {
		httpx.Fail(w, httpx.CodeUnauthorized, "账号不存在")
		return
	}

	nickname := admin.Nickname
	if nickname == "" {
		nickname = admin.Username
	}

	httpx.OK(w, map[string]interface{}{
		"userId":   admin.ID,
		"userName": admin.Username,
		"nickName": nickname,
		"email":    admin.Username + "@ar-musicapi.local",
		"avatar":   admin.Avatar,
		"roles":    admin.Roles,
		"buttons":  admin.Buttons,
	})
}

// profileRequest 资料更新请求体。
type profileRequest struct {
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
}

// handleUpdateProfile 修改昵称与头像。
func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request, claims *jwtx.Claims) {
	var req profileRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Fail(w, httpx.CodeBadRequest, "请求参数格式不正确")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	if err := s.store.UpdateAdminProfile(ctx, claims.UserID, req.Nickname, req.Avatar); err != nil {
		httpx.ServerError(w, "保存资料失败")
		return
	}
	httpx.OKMsg(w, "资料已更新", nil)
}
