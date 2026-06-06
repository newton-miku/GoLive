// Package handler 提供 HTTP 请求处理。
package handler

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"golive/internal/domain"
	"golive/internal/log"
	"golive/internal/service"
)

type authAPIResponse struct {
	Success bool          `json:"success"`
	Message string        `json:"message,omitempty"`
	Token   string        `json:"token,omitempty"`
	User    *authUserJSON `json:"user,omitempty"`
}

type authUserJSON struct {
	ID        any    `json:"id"` // int64，JSON 中序列化为数字
	Username  string `json:"username"`
	RoomID    string `json:"roomId"`
	StreamKey string `json:"streamKey,omitempty"`
}

// AuthAPIHandler 处理登录、注册、登出（JSON API）。
type AuthAPIHandler struct {
	userSvc *service.UserService
}

func NewAuthAPIHandler(userSvc *service.UserService) *AuthAPIHandler {
	return &AuthAPIHandler{userSvc: userSvc}
}

func (h *AuthAPIHandler) ServeHTTP(res http.ResponseWriter, req *http.Request) {
	res.Header().Set("Content-Type", "application/json")
	res.Header().Set("Access-Control-Allow-Origin", "*")

	path := strings.TrimPrefix(req.URL.Path, "/api/auth/")

	switch path {
	case "register":
		if req.Method == http.MethodOptions {
			return
		}
		if req.Method != http.MethodPost {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h.handleRegister(res, req)
	case "login":
		if req.Method == http.MethodOptions {
			return
		}
		if req.Method != http.MethodPost {
			res.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		h.handleLogin(res, req)
	case "logout":
		h.handleLogout(res, req)
	case "me":
		h.handleMe(res, req)
	case "reset-key":
		h.handleResetKey(res, req)
	default:
		http.NotFound(res, req)
	}
}

func (h *AuthAPIHandler) handleRegister(res http.ResponseWriter, req *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		res.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(res).Encode(authAPIResponse{Success: false, Message: "invalid request"})
		return
	}

	user, err := h.userSvc.Register(body.Username, body.Password)
	if err != nil {
		log.Debugf("[注册] 失败: username=%s err=%v\n", body.Username, err)
		switch {
		case errors.Is(err, service.ErrUserExists):
			res.WriteHeader(http.StatusConflict)
			json.NewEncoder(res).Encode(authAPIResponse{Success: false, Message: "用户名已被注册"})
		case errors.Is(err, service.ErrInvalidUsername):
			res.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(res).Encode(authAPIResponse{Success: false, Message: "用户名格式错误（3-20位字母数字下划线）"})
		case errors.Is(err, service.ErrInvalidPassword2):
			res.WriteHeader(http.StatusBadRequest)
			json.NewEncoder(res).Encode(authAPIResponse{Success: false, Message: "密码格式错误（6-32位）"})
		default:
			res.WriteHeader(http.StatusInternalServerError)
			json.NewEncoder(res).Encode(authAPIResponse{Success: false, Message: "注册失败"})
		}
		return
	}

	token, _ := service.GenerateToken(user)
	setSessionCookie(res, user.ID)

	// Get stream key for register response
	u := toAuthUser(user)
	streamKey, _ := h.userSvc.GetStreamKey(user.ID)
	if u != nil {
		u.StreamKey = streamKey
	}

	res.WriteHeader(http.StatusCreated)
	json.NewEncoder(res).Encode(authAPIResponse{
		Success: true,
		Message: "注册成功",
		Token:   token,
		User:    u,
	})
}

func (h *AuthAPIHandler) handleLogin(res http.ResponseWriter, req *http.Request) {
	var body struct {
		Username string `json:"username"`
		Password string `json:"password"`
	}
	if err := json.NewDecoder(req.Body).Decode(&body); err != nil {
		res.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(res).Encode(authAPIResponse{Success: false, Message: "invalid request"})
		return
	}

	user, err := h.userSvc.Login(body.Username, body.Password)
	if err != nil {
		if errors.Is(err, service.ErrUserNotFound) || errors.Is(err, service.ErrInvalidPassword) {
			res.WriteHeader(http.StatusUnauthorized)
			json.NewEncoder(res).Encode(authAPIResponse{Success: false, Message: "用户名或密码错误"})
			return
		}
		res.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(res).Encode(authAPIResponse{Success: false, Message: "登录失败"})
		return
	}

	token, _ := service.GenerateToken(user)
	setSessionCookie(res, user.ID)

	// Get stream key for login response
	u := toAuthUser(user)
	streamKey, _ := h.userSvc.GetStreamKey(user.ID)
	if u != nil {
		u.StreamKey = streamKey
	}

	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(authAPIResponse{
		Success: true,
		Message: "登录成功",
		Token:   token,
		User:    u,
	})
}

func (h *AuthAPIHandler) handleLogout(res http.ResponseWriter, req *http.Request) {
	clearSessionCookie(res)
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(authAPIResponse{Success: true, Message: "已退出"})
}

func (h *AuthAPIHandler) handleMe(res http.ResponseWriter, req *http.Request) {
	uid, ok := getSessionUID(req)
	if !ok {
		res.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(res).Encode(map[string]any{"loggedIn": false})
		return
	}

	user, err := h.userSvc.GetUserByID(uid)
	if err != nil || user == nil {
		clearSessionCookie(res)
		res.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(res).Encode(map[string]any{"loggedIn": false})
		return
	}

	res.WriteHeader(http.StatusOK)

	streamKey, _ := h.userSvc.GetStreamKey(uid)
	u := toAuthUser(user)
	if u != nil {
		u.StreamKey = streamKey
	}
	jwt, _ := service.GenerateToken(user)
	json.NewEncoder(res).Encode(map[string]any{
		"loggedIn": true,
		"token":    jwt,
		"user":     u,
	})
}

func (h *AuthAPIHandler) handleResetKey(res http.ResponseWriter, req *http.Request) {
	uid, ok := getSessionUID(req)
	if !ok {
		res.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(res).Encode(authAPIResponse{Success: false, Message: "未登录"})
		return
	}

	newKey, err := h.userSvc.ResetStreamKey(uid)
	if err != nil {
		res.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(res).Encode(authAPIResponse{Success: false, Message: "重置失败"})
		return
	}

	userForToken, _ := h.userSvc.GetUserByID(uid)
	token, _ := service.GenerateToken(userForToken)
	res.WriteHeader(http.StatusOK)
	json.NewEncoder(res).Encode(authAPIResponse{Success: true, Message: "密钥已重置", Token: token, User: &authUserJSON{StreamKey: newKey}})
}

func setSessionCookie(res http.ResponseWriter, uid int64) {
	http.SetCookie(res, &http.Cookie{
		Name:     "golive_sid",
		Value:    strconv.FormatInt(uid, 10),
		Path:     "/",
		HttpOnly: true,
		MaxAge:   30 * 24 * 3600, // 30天
		SameSite: http.SameSiteLaxMode,
	})
}

func clearSessionCookie(res http.ResponseWriter) {
	http.SetCookie(res, &http.Cookie{
		Name:   "golive_sid",
		Value:  "",
		Path:   "/",
		MaxAge: -1,
	})
}

// getSessionUID 从请求中获取当前登录用户 ID。返回 (id, ok)。
func getSessionUID(req *http.Request) (int64, bool) {
	auth := req.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		token := strings.TrimPrefix(auth, "Bearer ")
		uid, err := service.ValidateToken(token)
		if err != nil {
			return 0, false
		}
		return uid, true
	}
	cookie, err := req.Cookie("golive_sid")
	if err != nil {
		return 0, false
	}
	uid, err := strconv.ParseInt(cookie.Value, 10, 64)
	if err != nil {
		return 0, false
	}
	return uid, true
}

func toAuthUser(u *domain.User) *authUserJSON {
	if u == nil {
		return nil
	}
	return &authUserJSON{
		ID:       u.ID,
		Username: u.Username,
		RoomID:   u.RoomID,
	}
}

// GetSessionUIDFromRequest 从请求中获取当前登录用户 ID（供其他 handler 使用）。
func GetSessionUIDFromRequest(req *http.Request) (int64, bool) {
	return getSessionUID(req)
}
