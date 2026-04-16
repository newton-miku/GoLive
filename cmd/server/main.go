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
	"golive/internal/udpserver"
)

const (
	httpPort = 8080
	udpPort  = 8080 // WebRTC 媒体走同一 UDP 端口（TCP HTTP 信令不受影响）
)

func main() {
	// ===== 单 UDP 端口 WebRTC 服务 =====
	udpServer, err := udpserver.New(udpPort)
	if err != nil {
		fmt.Printf("UDP 端口 %d 监听失败: %v\n", udpPort, err)
		os.Exit(1)
	}
	defer udpServer.Close()
	fmt.Printf("[WebRTC] 单 UDP 端口 %d 已就绪\n", udpPort)

	// ===== 数据库 =====
	userRepo, err := repository.NewUserRepository("golive.db")
	if err != nil {
		fmt.Printf("数据库初始化失败: %v\n", err)
		os.Exit(1)
	}
	defer userRepo.Close()

	// ===== 依赖注入 =====
	roomRepo := repository.NewRoomRepository(userRepo.DB())
	streamSvc := service.NewStreamService(roomRepo)
	userSvc := service.NewUserService(userRepo, streamSvc)

	// 启动时从数据库加载所有房间，重建内存状态
	if err := roomRepo.LoadAll(); err != nil {
		fmt.Printf("加载直播间失败: %v\n", err)
	}
	users, _ := userRepo.ListAll()
	fmt.Printf("已恢复 %d 个直播间\n", len(users))

	// ===== HTTP 路由（信令） =====
	mux := http.NewServeMux()

	authAPI := handler.NewAuthAPIHandler(userSvc)
	mux.HandleFunc("/api/auth/login", authAPI.ServeHTTP)
	mux.HandleFunc("/api/auth/register", authAPI.ServeHTTP)
	mux.HandleFunc("/api/auth/logout", authAPI.ServeHTTP)
	mux.HandleFunc("/api/auth/me", authAPI.ServeHTTP)
	mux.HandleFunc("/api/auth/reset-key", authAPI.ServeHTTP)

	// WHIP / WHEP 使用共享 UDP 端口的 WebRTC API
	whipHandler := handler.NewWhipHandler(streamSvc, udpServer.API())
	mux.HandleFunc("/whip/", whipHandler.ServeHTTP)

	whepHandler := handler.NewWhepHandler(streamSvc, udpServer.API())
	mux.HandleFunc("/whep/", whepHandler.ServeHTTP)

	statusHandler := handler.NewStatusHandler(streamSvc)
	mux.HandleFunc("/status", statusHandler.ServeHTTP)
	mux.HandleFunc("/recommend", statusHandler.ServeHTTP)

	roomHandler := handler.NewRoomHandler(streamSvc)
	mux.HandleFunc("/rooms", roomHandler.ServeHTTP)

	// React SPA
	mux.HandleFunc("/live/", serveLive)

	frontendFS := http.FileServer(http.Dir("frontend/dist"))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		frontendFS.ServeHTTP(w, r)
	}))

	// ===== 启动 HTTP =====
	addr := fmt.Sprintf(":%d", httpPort)
	fmt.Println("=== GoLive 多直播间服务器 ===")
	fmt.Printf("信令地址: http://localhost%s\n", addr)
	fmt.Printf("媒体端口: UDP %d（所有 WebRTC 媒体走此端口）\n", udpPort)
	fmt.Println("================================")

	server := &http.Server{Addr: addr, Handler: mux}

	go func() {
		sig := make(chan os.Signal, 1)
		signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
		<-sig
		fmt.Println("\n正在关闭服务器...")
		udpServer.Close()
		server.Close()
	}()

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Printf("服务器错误: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("服务器已关闭")
}

func serveLive(w http.ResponseWriter, r *http.Request) {
	data, err := os.ReadFile("frontend/dist/index.html")
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write(data)
}
