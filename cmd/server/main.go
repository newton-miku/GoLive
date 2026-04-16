// Package main 是服务端入口。
package main

import (
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"golive/internal/handler"
	"golive/internal/repository"
	"golive/internal/service"
)

func main() {
	// ===== 数据库 =====
	userRepo, err := repository.NewUserRepository("golive.db")
	if err != nil {
		fmt.Printf("数据库初始化失败: %v\n", err)
		os.Exit(1)
	}
	defer userRepo.Close()

	// ===== 依赖注入 =====
	// 直播间表复用同一数据库连接
	roomRepo := repository.NewRoomRepository(userRepo.DB())
	streamSvc := service.NewStreamService(roomRepo)
	userSvc := service.NewUserService(userRepo, streamSvc)

	// 启动时从数据库加载所有房间，重建内存状态
	if err := roomRepo.LoadAll(); err != nil {
		fmt.Printf("加载直播间失败: %v\n", err)
	}
	users, _ := userRepo.ListAll()
	fmt.Printf("已恢复 %d 个直播间\n", len(users))

	// ===== HTTP 路由 =====
	mux := http.NewServeMux()

	// 用户认证（JSON API）
	authAPI := handler.NewAuthAPIHandler(userSvc)
	mux.HandleFunc("/api/auth/login", authAPI.ServeHTTP)
	mux.HandleFunc("/api/auth/register", authAPI.ServeHTTP)
	mux.HandleFunc("/api/auth/logout", authAPI.ServeHTTP)
	mux.HandleFunc("/api/auth/me", authAPI.ServeHTTP)
	mux.HandleFunc("/api/auth/reset-key", authAPI.ServeHTTP)

	// WHIP 推流
	whipHandler := handler.NewWhipHandler(streamSvc)
	mux.HandleFunc("/whip/", whipHandler.ServeHTTP)

	// WHEP 观看
	whepHandler := handler.NewWhepHandler(streamSvc)
	mux.HandleFunc("/whep/", whepHandler.ServeHTTP)

	// 状态和推荐
	statusHandler := handler.NewStatusHandler(streamSvc)
	mux.HandleFunc("/status", statusHandler.ServeHTTP)
	mux.HandleFunc("/recommend", statusHandler.ServeHTTP)

	// 房间管理
	roomHandler := handler.NewRoomHandler(streamSvc)
	mux.HandleFunc("/rooms", roomHandler.ServeHTTP)

	// 直播间页面（React SPA）
	mux.HandleFunc("/live/", serveLive)

	// 静态文件（React 前端 build 产物）
	frontendFS := http.FileServer(http.Dir("frontend/dist"))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// SPA fallback: 所有未匹配路径返回 index.html
		frontendFS.ServeHTTP(w, r)
	}))

	// ===== 启动 =====
	addr := ":8080"
	fmt.Println("=== GoLive 多直播间服务器 ===")
	fmt.Printf("访问地址: http://localhost%s\n", addr)
	fmt.Println("================================")

	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\n正在关闭服务器...")
		server.Close()
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("服务器错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("服务器已关闭")
}

// serveLive 处理 /live/{roomId} 路由，返回 React SPA。
func serveLive(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("frontend/dist/index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
