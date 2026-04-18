package handler

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"golive/internal/log"
	"golive/internal/service"

	"github.com/pion/webrtc/v4"
)

type WhepHandler struct {
	svc *service.StreamService
	api *webrtc.API
}

func NewWhepHandler(svc *service.StreamService, api *webrtc.API) *WhepHandler {
	return &WhepHandler{svc: svc, api: api}
}

func (h *WhepHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	roomID := strings.TrimPrefix(req.URL.Path, "/whep/")
	if roomID == "" {
		http.Error(res, "Room ID required", http.StatusBadRequest)
		return
	}

	log.Infof("[WHEP] 请求: room=%s method=%s\n", roomID, req.Method)

	if req.Method == http.MethodOptions {
		setCORS(res)
		return
	}

	room := h.svc.GetRoom(roomID)
	if room == nil {
		http.Error(res, "Room not found", http.StatusNotFound)
		return
	}
	if !room.IsActive() {
		http.Error(res, "Room is not streaming", http.StatusServiceUnavailable)
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
		log.Infof("[WHEP] 创建连接失败: %v\n", err)
		return
	}

	viewerVideoTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeH264, ClockRate: 90000},
		"video", "viewer")
	if err != nil {
		log.Infof("[WHEP] 创建视频轨道失败: %v\n", err)
		return
	}
	viewerAudioTrack, err := webrtc.NewTrackLocalStaticRTP(
		webrtc.RTPCodecCapability{MimeType: webrtc.MimeTypeOpus, ClockRate: 48000, Channels: 2},
		"audio", "viewer")
	if err != nil {
		log.Infof("[WHEP] 创建音频轨道失败: %v\n", err)
		return
	}

	_, err = peerConnection.AddTrack(viewerVideoTrack)
	if err != nil {
		log.Infof("[WHEP] AddTrack 视频失败: %v\n", err)
		return
	}
	_, err = peerConnection.AddTrack(viewerAudioTrack)
	if err != nil {
		log.Infof("[WHEP] AddTrack 音频失败: %v\n", err)
		return
	}

	room.AddViewerVideo(viewerVideoTrack)
	room.AddViewerAudio(viewerAudioTrack)

	peerConnection.OnTrack(func(track *webrtc.TrackRemote, receiver *webrtc.RTPReceiver) {
		log.Infof("[WHEP] 收到 OBS 轨道: kind=%s ssrc=0x%x\n", track.Kind(), track.SSRC())
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
		log.Infof("[WHEP] ICE: %s\n", state.String())
		if state == webrtc.ICEConnectionStateFailed || state == webrtc.ICEConnectionStateClosed {
			room.RemoveViewerVideo(viewerVideoTrack)
			room.RemoveViewerAudio(viewerAudioTrack)
		}
	})

	h.answer(res, peerConnection, offer, roomID)
}

func (h *WhepHandler) answer(res http.ResponseWriter, pc *webrtc.PeerConnection, offer []byte, roomID string) {
	if err := pc.SetRemoteDescription(webrtc.SessionDescription{
		Type: webrtc.SDPTypeOffer,
		SDP:  string(offer),
	}); err != nil {
		log.Infof("[WHEP] SetRemoteDescription 失败: %v\n", err)
		return
	}

	answer, err := pc.CreateAnswer(nil)
	if err != nil {
		log.Infof("[WHEP] CreateAnswer 失败: %v\n", err)
		return
	}
	if err = pc.SetLocalDescription(answer); err != nil {
		log.Infof("[WHEP] SetLocalDescription 失败: %v\n", err)
		return
	}

	select {
	case <-webrtc.GatheringCompletePromise(pc):
		log.Infof("[WHEP] ICE gathering 完成\n")
	case <-time.After(5 * time.Second):
		log.Infof("[WHEP] ICE gathering 超时\n")
	}

	sdp := pc.LocalDescription().SDP
	res.Header().Set("Location", "/whep/"+roomID)
	res.WriteHeader(http.StatusCreated)
	fmt.Fprint(res, sdp)
}
