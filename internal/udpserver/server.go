// Package udpserver 提供单 UDP 端口的 WebRTC 服务。
// 通过 pion-ice NewMultiUDPMuxFromPort，所有 PeerConnection 共享同一 UDP 端口收发 ICE/DTLS/RTP。
package udpserver

import (
	"sync"

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

// New 创建单 UDP 端口的 WebRTC 服务器。
// 所有通过返回 API 创建的 PeerConnection 都共享该 UDP 端口。
// HTTP 信令（WHIP/WHEP）仍走 TCP，由调用方另外处理。
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
