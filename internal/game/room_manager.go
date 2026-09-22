package game

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"log/slog"
	"sync"
)

type RoomManager struct {
	mu     sync.RWMutex
	rooms  map[string]*Room
	store  RecordStore
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	closed bool
}

func NewRoomManager(store RecordStore) *RoomManager {
	ctx, cancel := context.WithCancel(context.Background())
	return &RoomManager{rooms: make(map[string]*Room), store: store, ctx: ctx, cancel: cancel}
}
func (m *RoomManager) Create(a, b *Player) (*Room, error) {
	var id [16]byte
	if _, err := rand.Read(id[:]); err != nil {
		return nil, err
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed {
		return nil, context.Canceled
	}
	r := NewRoom(hex.EncodeToString(id[:]), a, b, m.store, m.remove)
	m.rooms[r.ID] = r
	a.setRoom(r)
	b.setRoom(r)
	m.wg.Add(1)
	go func() { defer m.wg.Done(); r.Run(m.ctx) }()
	slog.Info("Room created", "room_id", r.ID)
	return r, nil
}
func (m *RoomManager) Get(id string) *Room { m.mu.RLock(); defer m.mu.RUnlock(); return m.rooms[id] }
func (m *RoomManager) remove(r *Room) {
	m.mu.Lock()
	delete(m.rooms, r.ID)
	m.mu.Unlock()
	for _, p := range r.Players {
		p.clearRoom(r)
	}
}
func (m *RoomManager) Close() { m.mu.Lock(); m.closed = true; m.cancel(); m.mu.Unlock(); m.wg.Wait() }
