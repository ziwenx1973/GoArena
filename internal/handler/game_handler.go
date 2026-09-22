package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ziwenx1973/GoArena/internal/game"
	"github.com/ziwenx1973/GoArena/internal/service"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type GameHandler struct {
	Auth    *service.AuthService
	Matches *game.MatchMaker
	mu      sync.Mutex
	closed  bool
	wg      sync.WaitGroup
}

func (h *GameHandler) Connect(c *gin.Context) {
	claims, err := h.Auth.Parse(c.Query("token"))
	if err != nil {
		c.JSON(401, gin.H{"error": "invalid or expired token"})
		return
	}
	h.mu.Lock()
	if h.closed {
		h.mu.Unlock()
		c.JSON(503, gin.H{"error": "server shutting down"})
		return
	}
	h.wg.Add(1)
	h.mu.Unlock()
	defer h.wg.Done()
	upgrader := websocket.Upgrader{HandshakeTimeout: 5 * time.Second, CheckOrigin: func(r *http.Request) bool {
		origin := r.Header.Get("Origin")
		if origin == "" {
			return true
		}
		u, e := url.Parse(origin)
		return e == nil && (u.Scheme == "http" || u.Scheme == "https") && u.Host == r.Host
	}}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	p := game.NewPlayer(claims.UserID, claims.Username, conn)
	if err = h.Matches.Add(p); err != nil {
		conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
		_ = conn.WriteJSON(game.Message{Type: "error", Data: gin.H{"message": err.Error()}})
		p.Close()
		return
	}
	slog.Info("WebSocket connected", "user_id", p.UserID)
	writerDone := make(chan struct{})
	go func() { defer close(writerDone); p.WriteLoop() }()
	p.ReadLoop(h.Matches)
	h.Matches.Remove(p)
	<-writerDone
	slog.Info("WebSocket disconnected", "user_id", p.UserID)
}

// Shutdown also waits for upgraded connections, which http.Server does not track.
func (h *GameHandler) Shutdown() {
	h.mu.Lock()
	h.closed = true
	h.mu.Unlock()
	h.Matches.Close()
	h.wg.Wait()
}
