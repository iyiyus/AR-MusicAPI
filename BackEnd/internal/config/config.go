// Package config 负责加载与校验服务配置。
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"strconv"
)

// Config 是服务全量配置。
type Config struct {
	Server   ServerConfig   `json:"server"`
	Database DatabaseConfig `json:"database"`
	JWT      JWTConfig      `json:"jwt"`
	Admin    AdminConfig    `json:"admin"`
}

// ServerConfig HTTP 服务配置。
type ServerConfig struct {
	Port int    `json:"port"`
	Mode string `json:"mode"`
}

// DatabaseConfig MySQL 连接配置。
type DatabaseConfig struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	User     string `json:"user"`
	Password string `json:"password"`
	Name     string `json:"name"`
	Charset  string `json:"charset"`
}

// DSN 生成 go-sql-driver/mysql 所需的数据源名称。
func (d DatabaseConfig) DSN() string {
	return fmt.Sprintf(
		"%s:%s@tcp(%s:%d)/%s?charset=%s&parseTime=true&loc=Local&timeout=5s&readTimeout=15s&writeTimeout=15s",
		d.User, d.Password, d.Host, d.Port, d.Name, d.Charset,
	)
}

// JWTConfig 令牌配置。
type JWTConfig struct {
	Secret       string `json:"secret"`
	ExpireHours  int    `json:"expire_hours"`
	RefreshHours int    `json:"refresh_hours"`
}

// AdminConfig 首次启动时自动创建的管理员账号。
type AdminConfig struct {
	DefaultUsername string `json:"default_username"`
	DefaultPassword string `json:"default_password"`
}

// Load 读取配置文件，缺失项使用默认值，并支持环境变量覆盖。
func Load(path string) (*Config, error) {
	cfg := &Config{
		Server:   ServerConfig{Port: 8080, Mode: "debug"},
		Database: DatabaseConfig{Host: "127.0.0.1", Port: 3306, Charset: "utf8mb4"},
		JWT:      JWTConfig{ExpireHours: 24, RefreshHours: 168},
		Admin:    AdminConfig{DefaultUsername: "admin", DefaultPassword: "admin123"},
	}

	if data, err := os.ReadFile(path); err == nil {
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("解析配置文件 %s 失败: %w", path, err)
		}
	} else if !os.IsNotExist(err) {
		return nil, fmt.Errorf("读取配置文件 %s 失败: %w", path, err)
	}

	applyEnv(cfg)

	if cfg.Database.Name == "" {
		return nil, fmt.Errorf("配置项 database.name 不能为空")
	}
	if cfg.JWT.Secret == "" {
		return nil, fmt.Errorf("配置项 jwt.secret 不能为空")
	}
	if cfg.JWT.ExpireHours <= 0 {
		cfg.JWT.ExpireHours = 24
	}
	if cfg.JWT.RefreshHours <= 0 {
		cfg.JWT.RefreshHours = 168
	}
	if cfg.Server.Port <= 0 {
		cfg.Server.Port = 8080
	}
	return cfg, nil
}

// applyEnv 用 MUSIC_ 前缀的环境变量覆盖配置，便于容器化部署。
func applyEnv(cfg *Config) {
	if v := os.Getenv("MUSIC_SERVER_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Server.Port = n
		}
	}
	if v := os.Getenv("MUSIC_DB_HOST"); v != "" {
		cfg.Database.Host = v
	}
	if v := os.Getenv("MUSIC_DB_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			cfg.Database.Port = n
		}
	}
	if v := os.Getenv("MUSIC_DB_USER"); v != "" {
		cfg.Database.User = v
	}
	if v, ok := os.LookupEnv("MUSIC_DB_PASSWORD"); ok {
		cfg.Database.Password = v
	}
	if v := os.Getenv("MUSIC_DB_NAME"); v != "" {
		cfg.Database.Name = v
	}
	if v := os.Getenv("MUSIC_JWT_SECRET"); v != "" {
		cfg.JWT.Secret = v
	}
}
