package game

import (
	"encoding/json"
	"github.com/gorilla/websocket"
	"log/slog"
	"sync"
	"time"
)

type Player struct {
	UserID   uint
	Username string
	Conn     *websocket.Conn
	Send     chan Message
	Done     chan struct{}
	once     sync.Once
	mu       sync.RWMutex
	room     *Room
}

func NewPlayer(id uint, name string, conn *websocket.Conn) *Player {
	return &Player{UserID: id, Username: name, Conn: conn, Send: make(chan Message, 32), Done: make(chan struct{})}
}

// Send is deliberately never closed: multiple producers can enqueue safely.
// Done is closed exactly once and terminates both connection loops.
func (p *Player) Close() {
	p.once.Do(func() {
		close(p.Done)
		if p.Conn != nil {
			p.Conn.Close()
		}
	})
}
func (p *Player) Emit(m Message) {
	select {
	case <-p.Done:
		return
	default:
	}
	select {
	case p.Send <- m:
	case <-p.Done:
	default:
		p.Close()
	}
}
func (p *Player) Error(message string) {
	p.Emit(Message{"error", map[string]string{"message": message}})
}
func (p *Player) Room() *Room     { p.mu.RLock(); defer p.mu.RUnlock(); return p.room }
func (p *Player) setRoom(r *Room) { p.mu.Lock(); p.room = r; p.mu.Unlock() }
func (p *Player) clearRoom(r *Room) {
	p.mu.Lock()
	if p.room == r {
		p.room = nil
	}
	p.mu.Unlock()
}
func (p *Player) WriteLoop() {
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	defer p.Close()
	for {
		select {
		case <-p.Done:
			return
		case m := <-p.Send:
			p.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := p.Conn.WriteJSON(m); err != nil {
				return
			}
		case <-ticker.C:
			p.Conn.SetWriteDeadline(time.Now().Add(5 * time.Second))
			if err := p.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
func (p *Player) ReadLoop(mm *MatchMaker) {
	defer p.Close()
	p.Conn.SetReadLimit(4096)
	p.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	p.Conn.SetPongHandler(func(string) error { return p.Conn.SetReadDeadline(time.Now().Add(60 * time.Second)) })
	for {
		_, raw, err := p.Conn.ReadMessage()
		if err != nil {
			return
		}
		var m ClientMessage
		if json.Unmarshal(raw, &m) != nil {
			p.Error("invalid JSON")
			continue
		}
		switch m.Type {
		case "ping":
			p.Emit(Message{"pong", struct{}{}})
		case "start_match":
			if err := mm.Start(p); err != nil {
				p.Error(err.Error())
			}
		case "cancel_match":
			if err := mm.Cancel(p); err != nil {
				p.Error(err.Error())
			}
		case "action":
			var a struct {
				Choice Choice `json:"choice"`
				Round  int    `json:"round"`
			}
			if json.Unmarshal(m.Data, &a) != nil || !ValidChoice(a.Choice) {
				p.Error("choice must be rock, paper or scissors")
				continue
			}
			r := p.Room()
			if r == nil {
				p.Error("room not found")
				continue
			}
			if !r.Submit(GameEvent{p.UserID, a.Choice, a.Round}) {
				p.Error("room closed or busy")
			}
		default:
			p.Error("unknown message type")
		}
	}
}
func logRedis(err error) { slog.Error("Redis error", "error", err) }
