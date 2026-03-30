package web

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
)

type socketHub struct {
	mu      sync.RWMutex
	clients map[string]*socketClient
}

type socketClient struct {
	conn *websocket.Conn
	mu   sync.Mutex
}

func newSocketHub() *socketHub {
	return &socketHub{
		clients: map[string]*socketClient{},
	}
}

func (h *socketHub) register(clientID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.clients[clientID]; ok {
		_ = existing.conn.Close(websocket.StatusGoingAway, "replaced")
	}
	h.clients[clientID] = &socketClient{conn: conn}
}

func (h *socketHub) unregister(clientID string, conn *websocket.Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()

	existing, ok := h.clients[clientID]
	if !ok || existing.conn != conn {
		return
	}
	delete(h.clients, clientID)
}

func (h *socketHub) send(clientID string, event RenderSocketEvent) bool {
	h.mu.RLock()
	client := h.clients[clientID]
	h.mu.RUnlock()
	if client == nil {
		return false
	}

	client.mu.Lock()
	defer client.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := wsjson.Write(ctx, client.conn, event); err != nil {
		return false
	}
	return true
}

func (s *Server) handleSocket(w http.ResponseWriter, r *http.Request) {
	if !s.authorized(r) {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	clientID := strings.TrimSpace(r.URL.Query().Get("client_id"))
	if clientID == "" {
		http.Error(w, "missing client_id", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, nil)
	if err != nil {
		s.logf("ws accept client_id=%s err=%v", clientID, err)
		return
	}
	conn.SetReadLimit(1024)
	s.sockets.register(clientID, conn)
	defer s.sockets.unregister(clientID, conn)
	defer conn.Close(websocket.StatusNormalClosure, "")

	_ = s.sockets.send(clientID, RenderSocketEvent{
		Type:     "ready",
		ClientID: clientID,
		Message:  "socket connected",
	})

	readCtx := conn.CloseRead(r.Context())
	<-readCtx.Done()
	if err := readCtx.Err(); err != nil && !errors.Is(err, context.Canceled) {
		s.logf("ws close client_id=%s err=%v", clientID, err)
	}
}
