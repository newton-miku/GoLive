package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golive/internal/domain"
	"golive/internal/log"

	"github.com/pion/webrtc/v4"
)

type StreamServiceInterface interface {
	GetRoom(id string) *domain.Room
	OnStreamStart(room *domain.Room)
	OnStreamEnd(room *domain.Room)
}

type WhipHandler struct {
	svc StreamServiceInterface
	api *webrtc.API
}

func NewWhipHandler(svc StreamServiceInterface, api *webrtc.API) *WhipHandler {
	return &WhipHandler{svc: svc, api: api}
}

func (h *WhipHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	negotiated := false
	roomID := strings.TrimPrefix(req.URL.Path, "/whip/")
	if roomID == "" {
		http.Error(res, "Room ID required", http.StatusBadRequest)
		return
	}

	log.Infof("[WHIP] 请求: 房间=%s 方法=%s\n", roomID, req.Method)

	if req.Method == http.MethodOptions {
		setCORS(res)
		return
	}

	room := h.svc.GetRoom(roomID)
	if room == nil {
		http.Error(res, "Room not found", http.StatusNotFound)
		return
	}

	if req.Method == http.MethodDelete {
		h.svc.OnStreamEnd(room)
		log.Infof("[WHIP] 推流结束: %s\n", room.Name)
		return
	}

	auth := req.Header.Get("Authorization")
	password := strings.TrimPrefix(auth, "Bearer ")
	if password == "" || password != room.Password {
		log.Infof("[WHIP] 验证失败: %s\n", room.Name)
		http.Error(res, "Unauthorized", http.StatusUnauthorized)
		return
	}

	offer, err := io.ReadAll(req.Body)
	if err != nil {
		http.Error(res, "Failed to read offer", http.StatusBadRequest)
		return
	}

	setCORS(res)

	peerConnection, err := h.api.NewPeerConnection(webrtc.Configuration{
		ICEServers: []webrtc.ICEServer{
			{URLs: []string{"stun:stun.miwifi.com:3478"}},
			{URLs: []string{"stun:stun.chat.bilibili.com:3478"}},
			{URLs: []string{"stun:stun.douyucdn.cn:18000"}},
			{URLs: []string{"stun:stun.hitv.com:3478"}},
			{URLs: []string{"stun:stun.cloudflare.com:3478"}},
			{URLs: []string{"stun:stun.l.google.com:19302"}},
		},
	})
	if err != nil {
		log.Infof("[WHIP] 创建连接失败: %v\n", err)
		return
	}

	h.svc.OnStreamStart(room)
	log.Infof("[WHIP] 直播开始: %s\n", room.Name)

	defer func() {
		if !negotiated {
			h.svc.OnStreamEnd(room)
		}
	}()

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Infof("[WHIP] 收到轨道: kind=%s\n", track.Kind())

		go func() {
			for {
				_, _, err := receiver.ReadRTCP()
				if err != nil {
					return
				}
			}
		}()

		go func() {
			for {
				pkt, _, err := track.ReadRTP()
				if err != nil {
					return
				}
				room.WriteRTP(pkt, track.Kind())
			}
		}()
	})

	peerConnection.OnICEConnectionStateChange(func(state webrtc.ICEConnectionState) {
		log.Infof("[WHIP] ICE: %s\n", state.String())
		if state == webrtc.ICEConnectionStateFailed || state == webrtc.ICEConnectionStateClosed {
			h.svc.OnStreamEnd(room)
			peerConnection.Close()
		}
	})

	negotiated = h.answer(res, peerConnection, offer)
}

func (h *WhipHandler) answer(res http.ResponseWriter, pc *webrtc.PeerConnection, offer []byte) bool {
	if err := pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  string(offer),
	}); err != nil {
		log.Infof("[WHIP] SetRemoteDescription 失败: %v\n", err)
		return false
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Infof("[WHIP] CreateAnswer 失败: %v\n", err)
		return false
	}
	if err = pc.SetLocalDescription(answer); err != nil {
		log.Infof("[WHIP] SetLocalDescription 失败: %v\n", err)
		return false
	}

	select {
	case <-webrtc.GatheringCompletePromise(pc):
		log.Infof("[WHIP] ICE gathering 完成\n")
	case <-time.After(5 * time.Second):
		log.Infof("[WHIP] ICE gathering 超时\n")
	}

	sdp := pc.LocalDescription().SDP
	res.Header().Set("Location", "/whip/")
	res.WriteHeader(http.StatusCreated)
	fmt.Fprint(res, sdp)
	return true
}
