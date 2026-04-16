package handler

import (
	"encoding/json"
	"net/http"
	"time"

	"golive/internal/service"
)

// StatusHandler 处理状态查询和推荐列表。
type StatusHandler struct {
	svc *service.StreamService
}

func NewStatusHandler(svc *service.StreamService) *StatusHandler {
	return &StatusHandler{svc: svc}
}

func (h *StatusHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")

	if req.URL.Path == "/recommend" {
		h.serveRecommend(res)
		return
	}
	h.serveStatus(res, req)
}

func (h *StatusHandler) serveStatus(res http.ResponseWriter, req *http.Request) {
	roomID := req.URL.Query().Get("room")
	if roomID == "" {
		h.serveAllRooms(res)
		return
	}

	room := h.svc.GetRoom(roomID)
	if room == nil {
		res.WriteHeader(http.StatusNotFound)
		res.Write([]byte(`{"error":"room not found"}`))
		return
	}

	if room.IsActive() {
		viewers := room.GetViewerCount()
		startTime := room.GetStreamStartTime()
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		json.NewEncoder(res).Encode(map[string]any{
			"active":      true,
			"viewerCount": viewers,
			"uptime":      time.Since(startTime).Round(time.Second).String(),
		})
		return
	}

	res.WriteHeader(http.StatusServiceUnavailable)
	res.Write([]byte(`{"active":false}`))
}

func (h *StatusHandler) serveAllRooms(res http.ResponseWriter) {
	rooms := h.svc.ListRooms()
	result := make([]map[string]any, 0, len(rooms))
	for _, room := range rooms {
		result = append(result, map[string]any{
			"id":      room.ID,
			"name":    room.Name,
			"active":  room.IsActive(),
			"viewers": room.GetViewerCount(),
		})
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(result)
}

func (h *StatusHandler) serveRecommend(res http.ResponseWriter) {
	rooms := h.svc.RecommendRooms()
	result := make([]map[string]any, 0, len(rooms))
	for _, room := range rooms {
		result = append(result, map[string]any{
			"id":          room.ID,
			"name":        room.Name,
			"active":      room.IsActive(),
			"viewers":     room.GetViewerCount(),
			"streamStart": room.GetStreamStartTime().Unix(),
		})
	}
	res.Header().Set("Content-Type", "application/json")
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(result)
}

// RoomHandler 处理房间创建和管理。
type RoomHandler struct {
	svc *service.StreamService
}

func NewRoomHandler(svc *service.StreamService) *RoomHandler {
	return &RoomHandler{svc: svc}
}

func (h *RoomHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Access-Control-Allow-Origin", "*")

	if req.Method == http.MethodGet {
		// 返回房间列表（无密码）
		rooms := h.svc.ListRooms()
		result := make([]map[string]string, 0, len(rooms))
		for _, room := range rooms {
			result = append(result, map[string]string{
				"id":   room.ID,
				"name": room.Name,
			})
		}
		res.Header().Set("Content-Type", "application/json")
		res.WriteHeader(http.StatusOK)
		json.NewEncoder(res).Encode(result)
		return
	}

	if req.Method == http.MethodPost {
		var payload struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Password string `json:"password"`
		}
		if err := json.NewDecoder(req.Body).Decode(&payload); err != nil {
			http.Error(res, "Invalid request body", http.StatusBadRequest)
			return
		}
		if payload.ID == "" || payload.Name == "" || payload.Password == "" {
			http.Error(res, "Missing required fields", http.StatusBadRequest)
			return
		}
		if h.svc.GetRoom(payload.ID) != nil {
			http.Error(res, "Room already exists", http.StatusConflict)
			return
		}
		h.svc.CreateRoom(payload.ID, payload.Name, payload.Password)
		res.WriteHeader(http.StatusCreated)
		res.Write([]byte(`{"success":true}`))
		return
	}
}


