// Package udpserver 提供单 UDP 端口的 WebRTC 服务。
// 通过 pion-ice NewMultiUDPMuxFromPort，所有 PeerConnection 共享同一 UDP 端口收发 ICE/DTLS/RTP。
package udpserver

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/pion/ice/v4"
	"github.com/pion/logging"
	"github.com/pion/webrtc/v4"
)

// Server 单 UDP 端口的 WebRTC 服务器。
// 所有使用返回 API 创建的 PeerConnection 都通过同一个 UDP 端口收发媒体。
type Server struct {
	mux  *ice.MultiUDPMuxDefault
	api  *webrtc.API
	mu   sync.RWMutex
}

// DefaultPort 用于 WebRTC 媒体的默认 UDP 端口。
const DefaultPort = 8080

// detectExternalIP 通过多个外网服务自动检测出口 IP。
// 优先用 api.bilibili.com，兜底用 api.ipify.org。
func detectExternalIP() string {
	candidates := []struct {
		url    string
		parser func(body string) string
	}{
		{
			"https://api.bilibili.com/x/web-interface/zone",
			func(body string) string {
				// 响应结构: {"data":{"addr":"1.2.3.4",...}}
				i := strings.Index(body, `"addr"`)
				if i < 0 {
					return ""
				}
				start := strings.Index(body[i:], `"`) + i + 1
				end := strings.Index(body[start:], `"`) + start
				return strings.TrimSpace(body[start:end])
			},
		},
		{
			"https://api.ipify.org",
			func(body string) string {
				ip := strings.TrimSpace(body)
				if net.ParseIP(ip) != nil {
					return ip
				}
				return ""
			},
		},
	}

	for _, c := range candidates {
		ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
		req, _ := http.NewRequestWithContext(ctx, "GET", c.url, nil)
		if c.url == "https://api.bilibili.com/x/web-interface/zone" {
			req.Header.Set("User-Agent", "Mozilla/5.0")
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil || resp == nil {
			cancel()
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		resp.Body.Close()
		cancel()
		ip := c.parser(string(body))
		if net.ParseIP(ip) != nil {
			return ip
		}
	}
	return ""
}

// New 创建单 UDP 端口的 WebRTC 服务器。
// 所有通过返回 API 创建的 PeerConnection 都共享该 UDP 端口。
// HTTP 信令（WHIP/WHEP）仍走 TCP，由调用方另外处理。
//
// 支持 EXTERNAL_IP 环境变量：当存在外部 IP（如 Docker NAT 环境）时，
// 会将外部 IP 注册为 NAT1TO1 候选，使外部客户端能正确建立连接。
func New(port int) (*Server, error) {
	mux, err := ice.NewMultiUDPMuxFromPort(port,
		ice.UDPMuxFromPortWithLogger(logging.NewDefaultLoggerFactory().NewLogger("ice-mux")),
		ice.UDPMuxFromPortWithNetworks(ice.NetworkTypeUDP4),
	)
	if err != nil {
		return nil, err
	}

	se := webrtc.SettingEngine{}
	se.SetICEUDPMux(mux)

	// 注册 NAT1TO1 候选：外部 IP → 内部 0.0.0.0
	// 这样 Docker 等 NAT 环境下的 ICE 候选也能被外部客户端使用
	extIP := os.Getenv("EXTERNAL_IP")
	if extIP == "" {
		extIP = detectExternalIP()
	}
	if extIP != "" && net.ParseIP(extIP) != nil {
		se.SetICEAddressRewriteRules(webrtc.ICEAddressRewriteRule{
			External:        []string{extIP},
			Local:           "0.0.0.0",
			AsCandidateType: webrtc.ICECandidateTypeHost,
		})
		fmt.Printf("[WebRTC] NAT1TO1 外部 IP: %s\n", extIP)
	} else {
		fmt.Printf("[WebRTC] 未配置 EXTERNAL_IP，自动检测失败，将使用内网 ICE 候选\n")
	}

	api := webrtc.NewAPI(webrtc.WithSettingEngine(se))

	return &Server{mux: mux, api: api}, nil
}

// API 返回用于创建 PeerConnection 的 webrtc.API。
// 所有通过此 API 创建的 PeerConnection 都共享同一个 UDP 端口。
func (s *Server) API() *webrtc.API {
	return s.api
}

// Close 关闭所有底层 UDP 监听。
func (s *Server) Close() error {
	return s.mux.Close()
}
