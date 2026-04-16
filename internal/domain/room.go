package domain

import (
	"sync"
	"time"

	"github.com/pion/rtp"
	"github.com/pion/webrtc/v4"
)

type Room struct {
	ID       string
	Name     string
	Password string

	mu                  sync.RWMutex
	viewerVideoTracks   []*webrtc.TrackLocalStaticRTP
	viewerAudioTracks   []*webrtc.TrackLocalStaticRTP
	viewerCount         int

	streamMu        sync.RWMutex
	streamActive    bool
	streamStartTime time.Time
}

func NewRoom(id, name, password string) *Room {
	return &Room{
		ID:               id,
		Name:             name,
		Password:         password,
		viewerVideoTracks: []*webrtc.TrackLocalStaticRTP{},
		viewerAudioTracks: []*webrtc.TrackLocalStaticRTP{},
	}
}

func (r *Room) AddViewerVideo(t *webrtc.TrackLocalStaticRTP) {
	r.mu.Lock()
	r.viewerVideoTracks = append(r.viewerVideoTracks, t)
	r.viewerCount = len(r.viewerVideoTracks)
	r.mu.Unlock()
}

func (r *Room) AddViewerAudio(t *webrtc.TrackLocalStaticRTP) {
	r.mu.Lock()
	r.viewerAudioTracks = append(r.viewerAudioTracks, t)
	r.mu.Unlock()
}

func (r *Room) RemoveViewerVideo(t *webrtc.TrackLocalStaticRTP) {
	r.mu.Lock()
	for i, tr := range r.viewerVideoTracks {
		if tr == t {
			r.viewerVideoTracks = append(r.viewerVideoTracks[:i], r.viewerVideoTracks[i+1:]...)
			break
		}
	}
	r.viewerCount = len(r.viewerVideoTracks)
	r.mu.Unlock()
}

func (r *Room) RemoveViewerAudio(t *webrtc.TrackLocalStaticRTP) {
	r.mu.Lock()
	for i, tr := range r.viewerAudioTracks {
		if tr == t {
			r.viewerAudioTracks = append(r.viewerAudioTracks[:i], r.viewerAudioTracks[i+1:]...)
			break
		}
	}
	r.mu.Unlock()
}

func (r *Room) WriteRTP(pkt *rtp.Packet, kind webrtc.RTPCodecType) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	switch kind {
	case webrtc.RTPCodecTypeVideo:
		for _, t := range r.viewerVideoTracks {
			t.WriteRTP(pkt)
		}
	case webrtc.RTPCodecTypeAudio:
		for _, t := range r.viewerAudioTracks {
			t.WriteRTP(pkt)
		}
	}
}

func (r *Room) IsActive() bool {
	r.streamMu.RLock()
	defer r.streamMu.RUnlock()
	return r.streamActive
}

func (r *Room) GetViewerCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.viewerCount
}

func (r *Room) GetStreamStartTime() time.Time {
	r.streamMu.RLock()
	defer r.streamMu.RUnlock()
	return r.streamStartTime
}

func (r *Room) SetPassword(password string) {
	r.mu.Lock()
	r.Password = password
	r.mu.Unlock()
}

func (r *Room) SetStreamActive(active bool) {
	r.streamMu.Lock()
	r.streamActive = active
	if active {
		r.streamStartTime = time.Now()
	}
	r.streamMu.Unlock()
}
