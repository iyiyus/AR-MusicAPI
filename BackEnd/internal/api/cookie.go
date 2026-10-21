package api

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"time"

	"ar-musicapi/internal/httpx"
	"ar-musicapi/internal/jwtx"
	"ar-musicapi/internal/store"
)

// handleCookieList 分页查询 Cookie 池。
func (s *Server) handleCookieList(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	current, size := httpx.NormalizePage(
		httpx.QueryInt(r, "current", 1),
		httpx.QueryInt(r, "size", 20),
	)

	query := store.CookieQuery{
		Source:  httpx.QueryStr(r, "source"),
		Keyword: httpx.QueryStr(r, "keyword"),
		Current: current,
		Size:    size,
	}
	if raw := httpx.QueryStr(r, "status"); raw != "" {
		if n, err := strconv.Atoi(raw); err == nil && (n == 0 || n == 1) {
			query.Status = &n
		}
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	list, total, err := s.store.ListCookies(ctx, query)
	if err != nil {
		httpx.ServerError(w, "Cookie 列表查询失败")
		return
	}

	httpx.OK(w, map[string]interface{}{
		"records": list,
		"total":   total,
		"current": current,
		"size":    size,
	})
}

// cookieAddRequest 批量新增 Cookie。
type cookieAddRequest struct {
	Source string   `json:"source"`
	Remark string   `json:"remark"`
	Cookie string   `json:"cookie"`
	List   []string `json:"cookies"`
}

// handleCookieAdd 新增 Cookie，支持单条与批量。
func (s *Server) handleCookieAdd(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	var req cookieAddRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Fail(w, httpx.CodeBadRequest, "请求参数格式不正确")
		return
	}
	if strings.TrimSpace(req.Source) == "" {
		httpx.Fail(w, httpx.CodeBadRequest, "请选择音乐源")
		return
	}
	if _, ok := s.registry.Get(req.Source); !ok {
		httpx.Fail(w, httpx.CodeBadRequest, "不支持的音乐源")
		return
	}

	raw := req.List
	if strings.TrimSpace(req.Cookie) != "" {
		raw = append(raw, req.Cookie)
	}
	if len(raw) == 0 {
		httpx.Fail(w, httpx.CodeBadRequest, "请填写 Cookie 内容")
		return
	}

	items := make([]store.Cookie, 0, len(raw))
	for _, item := range raw {
		if strings.TrimSpace(item) == "" {
			continue
		}
		items = append(items, store.Cookie{
			Source: req.Source,
			Cookie: item,
			Remark: req.Remark,
			Status: 1,
		})
	}
	if len(items) == 0 {
		httpx.Fail(w, httpx.CodeBadRequest, "没有有效的内容")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	count, err := s.store.AddCookies(ctx, items)
	if err != nil {
		httpx.ServerError(w, "Cookie 保存失败")
		return
	}

	httpx.OKMsg(w, "成功添加 "+strconv.Itoa(count)+" 条 Cookie", map[string]interface{}{"count": count})
}

// cookieUpdateRequest 修改单条 Cookie。
type cookieUpdateRequest struct {
	ID     uint   `json:"id"`
	Source string `json:"source"`
	Cookie string `json:"cookie"`
	Remark string `json:"remark"`
	Status int    `json:"status"`
}

// handleCookieUpdate 修改 Cookie 内容或状态。
func (s *Server) handleCookieUpdate(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	var req cookieUpdateRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Fail(w, httpx.CodeBadRequest, "请求参数格式不正确")
		return
	}
	if req.ID == 0 {
		httpx.Fail(w, httpx.CodeBadRequest, "缺少记录 ID")
		return
	}
	if req.Source != "" {
		if _, ok := s.registry.Get(req.Source); !ok {
			httpx.Fail(w, httpx.CodeBadRequest, "不支持的音乐源")
			return
		}
	}
	if strings.TrimSpace(req.Cookie) == "" {
		httpx.Fail(w, httpx.CodeBadRequest, "Cookie 内容不能为空")
		return
	}
	if req.Status != 0 {
		req.Status = 1
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err := s.store.UpdateCookie(ctx, store.Cookie{
		ID:     req.ID,
		Source: req.Source,
		Cookie: req.Cookie,
		Remark: req.Remark,
		Status: req.Status,
	})
	if err != nil {
		if err == store.ErrNotFound {
			httpx.Fail(w, httpx.CodeNotFound, "记录不存在")
			return
		}
		httpx.ServerError(w, "Cookie 更新失败")
		return
	}
	httpx.OKMsg(w, "保存成功", nil)
}

// parseIDs 解析 ids=1,2,3 或 id=1。
func parseIDs(r *http.Request) []uint {
	raw := httpx.QueryStr(r, "ids")
	if raw == "" {
		raw = httpx.QueryStr(r, "id")
	}
	if raw == "" {
		return nil
	}

	parts := strings.Split(raw, ",")
	out := make([]uint, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		if n, err := strconv.ParseUint(p, 10, 32); err == nil {
			out = append(out, uint(n))
		}
	}
	return out
}

// idsRequest 批量操作请求体。
type idsRequest struct {
	IDs []uint `json:"ids"`
	ID  uint   `json:"id"`
}

func (req *idsRequest) all() []uint {
	out := make([]uint, 0, len(req.IDs)+1)
	out = append(out, req.IDs...)
	if req.ID != 0 {
		out = append(out, req.ID)
	}
	return out
}

// handleCookieDelete 批量删除。
func (s *Server) handleCookieDelete(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	ids := parseIDs(r)
	if len(ids) == 0 {
		httpx.Fail(w, httpx.CodeBadRequest, "请选择要删除的记录")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	n, err := s.store.DeleteCookies(ctx, ids)
	if err != nil {
		httpx.ServerError(w, "删除失败")
		return
	}
	httpx.OKMsg(w, "已删除 "+strconv.FormatInt(n, 10)+" 条记录", map[string]interface{}{"count": n})
}

// handleCookieToggle 批量启用/禁用。
func (s *Server) handleCookieToggle(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	var req idsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Fail(w, httpx.CodeBadRequest, "请求参数格式不正确")
		return
	}
	ids := req.all()
	if len(ids) == 0 {
		httpx.Fail(w, httpx.CodeBadRequest, "请选择要操作的记录")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	n, err := s.store.ToggleCookies(ctx, ids)
	if err != nil {
		httpx.ServerError(w, "状态切换失败")
		return
	}
	httpx.OKMsg(w, "已更新 "+strconv.FormatInt(n, 10)+" 条记录", map[string]interface{}{"count": n})
}

// handleCookieReset 批量重置失败计数。
func (s *Server) handleCookieReset(w http.ResponseWriter, r *http.Request, _ *jwtx.Claims) {
	var req idsRequest
	if err := httpx.DecodeJSON(r, &req); err != nil {
		httpx.Fail(w, httpx.CodeBadRequest, "请求参数格式不正确")
		return
	}
	ids := req.all()
	if len(ids) == 0 {
		httpx.Fail(w, httpx.CodeBadRequest, "请选择要操作的记录")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	n, err := s.store.ResetCookies(ctx, ids)
	if err != nil {
		httpx.ServerError(w, "重置失败")
		return
	}
	httpx.OKMsg(w, "已重置 "+strconv.FormatInt(n, 10)+" 条记录", map[string]interface{}{"count": n})
}
