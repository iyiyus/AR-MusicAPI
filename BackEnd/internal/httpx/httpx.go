// Package httpx 统一 JSON 响应格式与请求解析。
//
// 约定：所有 /api 接口始终返回 HTTP 200，业务状态放在 code 字段，
// 与前端 axios 封装（响应拦截器读取 code）保持一致。
package httpx

import (
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strconv"
	"strings"
)

// 业务状态码，与前端 ApiStatus 对齐。
const (
	CodeSuccess      = 200
	CodeBadRequest   = 400
	CodeUnauthorized = 401
	CodeForbidden    = 403
	CodeNotFound     = 404
	CodeServerError  = 500
)

// Response 统一响应体。
type Response struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// OK 返回成功响应。
func OK(w http.ResponseWriter, data interface{}) {
	write(w, Response{Code: CodeSuccess, Msg: "success", Data: data})
}

// OKMsg 返回带自定义提示的成功响应。
func OKMsg(w http.ResponseWriter, msg string, data interface{}) {
	write(w, Response{Code: CodeSuccess, Msg: msg, Data: data})
}

// Fail 返回业务失败响应。
func Fail(w http.ResponseWriter, code int, msg string) {
	write(w, Response{Code: code, Msg: msg})
}

// ServerError 返回服务端错误。
func ServerError(w http.ResponseWriter, msg string) {
	write(w, Response{Code: CodeServerError, Msg: msg})
}

func write(w http.ResponseWriter, resp Response) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

// DecodeJSON 解析请求体 JSON，限制 2MB。
func DecodeJSON(r *http.Request, dst interface{}) error {
	if r.Body == nil {
		return errors.New("请求体为空")
	}
	dec := json.NewDecoder(io.LimitReader(r.Body, 2<<20))
	if err := dec.Decode(dst); err != nil {
		if errors.Is(err, io.EOF) {
			return errors.New("请求体为空")
		}
		return err
	}
	return nil
}

// QueryInt 读取查询参数中的整数，缺省返回 def。
func QueryInt(r *http.Request, key string, def int) int {
	v := r.URL.Query().Get(key)
	if v == "" {
		return def
	}
	n, err := strconv.Atoi(v)
	if err != nil {
		return def
	}
	return n
}

// QueryStr 读取查询参数中的字符串（已 trim）。
func QueryStr(r *http.Request, key string) string {
	return strings.TrimSpace(r.URL.Query().Get(key))
}

// NormalizePage 归一化分页参数。
func NormalizePage(current, size int) (int, int) {
	if current < 1 {
		current = 1
	}
	if size < 1 {
		size = 20
	}
	if size > 200 {
		size = 200
	}
	return current, size
}

// ClientIP 尽力获取真实客户端 IP。
func ClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		if idx := strings.IndexByte(xff, ','); idx > 0 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}
	if xr := r.Header.Get("X-Real-IP"); xr != "" {
		return strings.TrimSpace(xr)
	}
	host := r.RemoteAddr
	if idx := strings.LastIndex(host, ":"); idx > 0 {
		host = host[:idx]
	}
	return strings.Trim(host, "[]")
}
