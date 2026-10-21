// AR-musicAPI 后端服务入口。
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"ar-musicapi/internal/api"
	"ar-musicapi/internal/config"
	"ar-musicapi/internal/music"
	"ar-musicapi/internal/store"
	"golang.org/x/crypto/bcrypt"
)

func main() {
	configPath := flag.String("config", "config.json", "配置文件路径")
	flag.Parse()

	log.SetFlags(log.LstdFlags)

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("[fatal] 配置加载失败: %v", err)
	}

	st, err := store.Open(cfg.Database.DSN())
	if err != nil {
		log.Fatalf("[fatal] 数据库连接失败: %v\n请确认 MySQL 已启动，且 %s 库已按 db/schema.sql 初始化",
			err, cfg.Database.Name)
	}
	defer func() { _ = st.Close() }()

	bootCtx, bootCancel := context.WithTimeout(context.Background(), 15*time.Second)
	if err := bootstrap(bootCtx, cfg, st); err != nil {
		bootCancel()
		log.Fatalf("[fatal] 初始化失败: %v", err)
	}
	bootCancel()

	registry := music.NewRegistry()
	server := api.New(cfg, st, registry)

	httpServer := &http.Server{
		Addr:              fmt.Sprintf(":%d", cfg.Server.Port),
		Handler:           server.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
		ReadTimeout:       30 * time.Second,
		WriteTimeout:      30 * time.Second,
		IdleTimeout:       60 * time.Second,
	}

	go func() {
		log.Printf("[info] AR-musicAPI 已启动 -> http://127.0.0.1:%d", cfg.Server.Port)
		log.Printf("[info] 健康检查     -> http://127.0.0.1:%d/api/health", cfg.Server.Port)
		log.Printf("[info] 点歌接口     -> http://127.0.0.1:%d/api/music/search?name=晴天", cfg.Server.Port)
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[fatal] HTTP 服务启动失败: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	log.Println("[info] 正在关闭服务...")
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		log.Printf("[warn] 优雅关闭超时: %v", err)
	}
	log.Println("[info] 服务已停止")
}

// bootstrap 首次启动时创建默认管理员。
func bootstrap(ctx context.Context, cfg *config.Config, st *store.Store) error {
	n, err := st.CountAdmins(ctx)
	if err != nil {
		return err
	}
	if n > 0 {
		return nil
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(cfg.Admin.DefaultPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}
	if err := st.CreateAdmin(ctx, cfg.Admin.DefaultUsername, string(hash),
		"超级管理员", "R_SUPER", "add,edit,delete,reset"); err != nil {
		return err
	}

	log.Printf("[info] 已创建默认管理员: %s / %s （请登录后立即修改密码）",
		cfg.Admin.DefaultUsername, cfg.Admin.DefaultPassword)
	return nil
}
