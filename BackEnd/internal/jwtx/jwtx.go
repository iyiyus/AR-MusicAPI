// Package jwtx 实现最小化的 HS256 JWT 签发与校验（仅依赖标准库）。
package jwtx

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

// 令牌用途。
const (
	TypeAccess  = "access"
	TypeRefresh = "refresh"
)

var (
	// ErrInvalidToken 令牌格式或签名不正确。
	ErrInvalidToken = errors.New("无效的令牌")
	// ErrExpiredToken 令牌已过期。
	ErrExpiredToken = errors.New("令牌已过期")
)

// Claims 令牌载荷。
type Claims struct {
	UserID   uint     `json:"uid"`
	UserName string   `json:"uname"`
	Roles    []string `json:"roles"`
	TokenTyp string   `json:"typ"`
	Iat      int64    `json:"iat"`
	Exp      int64    `json:"exp"`
}

// Manager 签发与解析令牌。
type Manager struct {
	secret []byte
}

// NewManager 创建令牌管理器。
func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

var headerSegment = base64.RawURLEncoding.EncodeToString([]byte(`{"alg":"HS256","typ":"JWT"}`))

// Issue 签发令牌。
func (m *Manager) Issue(userID uint, userName string, roles []string, ttl time.Duration, tokenType string) (string, error) {
	now := time.Now()
	claims := Claims{
		UserID:   userID,
		UserName: userName,
		Roles:    roles,
		TokenTyp: tokenType,
		Iat:      now.Unix(),
		Exp:      now.Add(ttl).Unix(),
	}
	payload, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	signing := headerSegment + "." + base64.RawURLEncoding.EncodeToString(payload)

	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(signing))
	signature := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))

	return signing + "." + signature, nil
}

// Parse 校验并解析令牌。
func (m *Manager) Parse(token string) (*Claims, error) {
	token = strings.TrimSpace(token)
	token = strings.TrimPrefix(token, "Bearer ")
	token = strings.TrimSpace(token)

	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return nil, ErrInvalidToken
	}

	mac := hmac.New(sha256.New, m.secret)
	mac.Write([]byte(parts[0] + "." + parts[1]))
	expected := base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(parts[2])) {
		return nil, ErrInvalidToken
	}

	raw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return nil, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return nil, ErrInvalidToken
	}
	if claims.Exp > 0 && time.Now().Unix() >= claims.Exp {
		return nil, ErrExpiredToken
	}
	return &claims, nil
}
