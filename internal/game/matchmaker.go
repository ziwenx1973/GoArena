package game

import (
	"context"
	"errors"
	"fmt"
	"github.com/redis/go-redis/v9"
	"log/slog"
	"strconv"
	"sync"
	"time"
)

const queueKey = "match:queue"

type MatchMaker struct {
	mu      sync.Mutex
	redis   *redis.Client
	rooms   *RoomManager
	players map[uint]*Player
	queued  map[uint]bool
	closed  bool
	stop    chan struct{}
	done    chan struct{}
}

func NewMatchMaker(r *redis.Client, rooms *RoomManager) (*MatchMaker, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Single server owns this queue; rooms cannot survive process restart.
	if err := r.Del(ctx, queueKey).Err(); err != nil {
		return nil, err
	}
	m := &MatchMaker{redis: r, rooms: rooms, players: make(map[uint]*Player), queued: make(map[uint]bool), stop: make(chan struct{}), done: make(chan struct{})}
	go m.run()
	return m, nil
}
func onlineKey(id uint) string { return fmt.Sprintf("online:user:%d", id) }
func (m *MatchMaker) Add(p *Player) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return errors.New("server shutting down")
	}
	if m.players[p.UserID] != nil {
		return errors.New("user already connected")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := m.redis.Set(ctx, onlineKey(p.UserID), p.Username, 90*time.Second).Err(); err != nil {
		logRedis(err)
		return errors.New("online service unavailable")
	}
	m.players[p.UserID] = p
	return nil
}
func (m *MatchMaker) Remove(p *Player) {
	p.Close()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.players[p.UserID] != p {
		return
	}
	delete(m.players, p.UserID)
	delete(m.queued, p.UserID)
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pipe := m.redis.Pipeline()
	pipe.LRem(ctx, queueKey, 0, p.UserID)
	pipe.Del(ctx, onlineKey(p.UserID))
	if _, err := pipe.Exec(ctx); err != nil {
		logRedis(err)
	}
}
func (m *MatchMaker) Start(p *Player) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed || m.players[p.UserID] != p {
		return errors.New("connection unavailable")
	}
	if p.Room() != nil {
		return errors.New("already in a room")
	}
	if m.queued[p.UserID] {
		return errors.New("already matchmaking")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	// Remove any stale occurrence before appending, including after an uncertain write.
	pipe := m.redis.TxPipeline()
	pipe.LRem(ctx, queueKey, 0, p.UserID)
	pipe.RPush(ctx, queueKey, p.UserID)
	if _, err := pipe.Exec(ctx); err != nil {
		logRedis(err)
		return errors.New("matchmaking unavailable; retry")
	}
	m.queued[p.UserID] = true
	p.Emit(Message{"match_waiting", struct{}{}})
	slog.Info("Player enters matchmaking", "user_id", p.UserID)
	return nil
}
func (m *MatchMaker) Cancel(p *Player) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.players[p.UserID] != p {
		return errors.New("connection unavailable")
	}
	if p.Room() != nil {
		return errors.New("already in a room")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	if err := m.redis.LRem(ctx, queueKey, 0, p.UserID).Err(); err != nil {
		logRedis(err)
		return errors.New("cannot cancel now; retry")
	}
	delete(m.queued, p.UserID)
	p.Emit(Message{"match_cancelled", struct{}{}})
	slog.Info("Player cancels matchmaking", "user_id", p.UserID)
	return nil
}
func (m *MatchMaker) run() {
	defer close(m.done)
	tick := time.NewTicker(200 * time.Millisecond)
	defer tick.Stop()
	heartbeat := time.NewTicker(30 * time.Second)
	defer heartbeat.Stop()
	for {
		select {
		case <-m.stop:
			return
		case <-tick.C:
			m.match()
		case <-heartbeat.C:
			m.refresh()
		}
	}
}
func (m *MatchMaker) match() {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	ids, err := m.redis.LRange(ctx, queueKey, 0, 1).Result()
	if err != nil {
		logRedis(err)
		return
	}
	if len(ids) == 0 {
		return
	}
	ps := make([]*Player, 0, 2)
	for _, s := range ids {
		id, e := strconv.ParseUint(s, 10, 64)
		p := m.players[uint(id)]
		if e != nil || p == nil || !m.queued[uint(id)] || p.Room() != nil {
			if err := m.redis.LRem(ctx, queueKey, 0, s).Err(); err != nil {
				logRedis(err)
			}
			return
		}
		select {
		case <-p.Done:
			delete(m.queued, p.UserID)
			return
		default:
		}
		ps = append(ps, p)
	}
	if len(ps) < 2 {
		return
	}
	if err := m.redis.LTrim(ctx, queueKey, 2, -1).Err(); err != nil {
		logRedis(err)
		for _, p := range ps {
			delete(m.queued, p.UserID)
			p.Error("matchmaking interrupted; start again")
		}
		return
	}
	for _, p := range ps {
		delete(m.queued, p.UserID)
	}
	r, err := m.rooms.Create(ps[0], ps[1])
	if err != nil {
		for _, p := range ps {
			p.Error("cannot create room; retry")
		}
		return
	}
	slog.Info("Match success", "room_id", r.ID)
}
func (m *MatchMaker) refresh() {
	m.mu.Lock()
	defer m.mu.Unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pipe := m.redis.Pipeline()
	for _, p := range m.players {
		pipe.Set(ctx, onlineKey(p.UserID), p.Username, 90*time.Second)
	}
	if _, err := pipe.Exec(ctx); err != nil {
		logRedis(err)
	}
}
func (m *MatchMaker) Close() {
	m.mu.Lock()
	if !m.closed {
		m.closed = true
		close(m.stop)
	}
	players := make([]*Player, 0, len(m.players))
	for _, p := range m.players {
		players = append(players, p)
	}
	m.mu.Unlock()
	<-m.done
	m.rooms.Close()
	for _, p := range players {
		m.Remove(p)
	}
}
