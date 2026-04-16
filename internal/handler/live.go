package handler

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"

	"golive/internal/domain"
	"golive/internal/service"
)

// LiveHandler 渲染直播间页面。
type LiveHandler struct {
	tmpl     *template.Template
	svc      *service.StreamService
	userSvc  *service.UserService
}

func NewLiveHandler(tmpl *template.Template, svc *service.StreamService, userSvc *service.UserService) *LiveHandler {
	return &LiveHandler{tmpl: tmpl, svc: svc, userSvc: userSvc}
}

func (h *LiveHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	roomID := strings.TrimPrefix(req.URL.Path, "/live/")
	if roomID == "" || roomID == "/" {
		http.Redirect(res, req, "/", http.StatusFound)
		return
	}

	room := h.svc.GetRoom(roomID)
	if room == nil {
		room = domain.NewRoom(roomID, roomID, "未配置")
	}

	// 获取当前登录用户
	var currentUser *domain.User
	if uid, ok := GetSessionUIDFromRequest(req); ok && h.userSvc != nil {
		currentUser, _ = h.userSvc.GetUserByID(uid)
	}

	host := req.Host
	if req.TLS == nil {
		host = "http://" + host
	} else {
		host = "https://" + host
	}

	// 是否是房主
	isOwner := currentUser != nil && currentUser.RoomID == roomID

	data := map[string]any{
		"Room":       room,
		"PushURL":    fmt.Sprintf("%s/whip/%s", host, roomID),
		"PlayURL":    fmt.Sprintf("%s/live/%s", host, roomID),
		"Host":       host,
		"CurrentUser": currentUser,
		"IsOwner":    isOwner,
	}

	res.Header().Set("Content-Type", "text/html; charset=utf-8")
	h.tmpl.ExecuteTemplate(res, "live", data)
}
