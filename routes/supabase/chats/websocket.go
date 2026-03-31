package chats

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type wsEvent struct {
	Type   string                 `json:"type"`
	ChatID int64                  `json:"chat_id,omitempty"`
	Data   map[string]interface{} `json:"data,omitempty"`
}

type chatWSHub struct {
	mu    sync.RWMutex
	conns map[string]map[*websocket.Conn]struct{}
}

func newChatWSHub() *chatWSHub {
	return &chatWSHub{
		conns: make(map[string]map[*websocket.Conn]struct{}),
	}
}

func (h *chatWSHub) addConn(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.conns[userID]; !ok {
		h.conns[userID] = make(map[*websocket.Conn]struct{})
	}
	h.conns[userID][conn] = struct{}{}
}

func (h *chatWSHub) removeConn(userID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	userConns, ok := h.conns[userID]
	if !ok {
		return
	}
	delete(userConns, conn)
	if len(userConns) == 0 {
		delete(h.conns, userID)
	}
}

func (h *chatWSHub) emitToUsers(userIDs []string, event wsEvent) {
	payload, err := json.Marshal(event)
	if err != nil {
		return
	}

	seen := make(map[string]struct{})
	for _, userID := range userIDs {
		if strings.TrimSpace(userID) == "" {
			continue
		}
		if _, alreadySeen := seen[userID]; alreadySeen {
			continue
		}
		seen[userID] = struct{}{}

		h.mu.RLock()
		userConns := h.conns[userID]
		targets := make([]*websocket.Conn, 0, len(userConns))
		for c := range userConns {
			targets = append(targets, c)
		}
		h.mu.RUnlock()

		for _, conn := range targets {
			_ = conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := conn.WriteMessage(websocket.TextMessage, payload); err != nil {
				_ = conn.Close()
				h.removeConn(userID, conn)
			}
		}
	}
}

var chatRealtimeHub = newChatWSHub()

var chatWSUpgrader = websocket.Upgrader{
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *ChatHandler) ChatEventsWS(w http.ResponseWriter, r *http.Request) {
	userID := strings.TrimSpace(r.URL.Query().Get("user_id"))
	if !isValidSixDigitUserID(userID) {
		http.Error(w, "missing or invalid user_id", http.StatusUnauthorized)
		return
	}

	conn, err := chatWSUpgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}

	chatRealtimeHub.addConn(userID, conn)
	defer func() {
		chatRealtimeHub.removeConn(userID, conn)
		_ = conn.Close()
	}()

	for {
		if _, _, err := conn.ReadMessage(); err != nil {
			return
		}
	}
}

