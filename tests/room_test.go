package tests

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/ziwenx1973/GoArena/internal/game"
	"github.com/ziwenx1973/GoArena/internal/model"
	"sync"
	"testing"
	"time"
)

type memoryStore struct {
	mu      sync.Mutex
	records []model.GameRecord
	err     error
}

func (s *memoryStore) Save(_ context.Context, r *model.GameRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return s.err
	}
	s.records = append(s.records, *r)
	return nil
}
func receive(t *testing.T, p *game.Player, kind string) game.Message {
	t.Helper()
	select {
	case m := <-p.Send:
		if m.Type != kind {
			t.Fatalf("got %s want %s: %+v", m.Type, kind, m.Data)
		}
		return m
	case <-time.After(3 * time.Second):
		t.Fatalf("timed out waiting for %s", kind)
	}
	return game.Message{}
}
func waitDone(t *testing.T, r *game.Room) {
	t.Helper()
	select {
	case <-r.Done:
	case <-time.After(3 * time.Second):
		t.Fatal("room goroutine did not exit")
	}
}
func TestRoomBestOfThree(t *testing.T) {
	store := &memoryStore{}
	manager := game.NewRoomManager(store)
	defer manager.Close()
	a, b := game.NewPlayer(1, "alice", nil), game.NewPlayer(2, "bob", nil)
	r, err := manager.Create(a, b)
	if err != nil {
		t.Fatal(err)
	}
	receive(t, a, "match_success")
	receive(t, b, "match_success")
	r.Submit(game.GameEvent{PlayerID: 1, Choice: "invalid"})
	receive(t, a, "error")
	r.Submit(game.GameEvent{PlayerID: 1, Choice: game.Rock, Round: 1})
	r.Submit(game.GameEvent{PlayerID: 1, Choice: game.Paper, Round: 1})
	receive(t, a, "error")
	r.Submit(game.GameEvent{PlayerID: 2, Choice: game.Rock, Round: 1})
	receive(t, a, "round_result")
	receive(t, b, "round_result")
	r.Submit(game.GameEvent{PlayerID: 1, Choice: game.Rock, Round: 1})
	receive(t, a, "error")
	for round := 2; round <= 3; round++ {
		r.Submit(game.GameEvent{PlayerID: 1, Choice: game.Rock, Round: round})
		r.Submit(game.GameEvent{PlayerID: 2, Choice: game.Scissors, Round: round})
		receive(t, a, "round_result")
		receive(t, b, "round_result")
	}
	receive(t, a, "game_over")
	receive(t, b, "game_over")
	waitDone(t, r)
	if manager.Get(r.ID) != nil || a.Room() != nil || b.Room() != nil {
		t.Fatal("room not cleaned up")
	}
	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.records) != 1 {
		t.Fatalf("records: %+v", store.records)
	}
	v := store.records[0]
	if v.WinnerID == nil || *v.WinnerID != 1 || v.Player1Score != 2 || v.Player2Score != 0 {
		t.Fatalf("wrong result: %+v", v)
	}
}
func TestRoomDisconnectAndSaveFailure(t *testing.T) {
	for _, failure := range []bool{false, true} {
		t.Run(map[bool]string{false: "saved", true: "failed"}[failure], func(t *testing.T) {
			store := &memoryStore{}
			if failure {
				store.err = errors.New("storage unavailable")
			}
			a, b := game.NewPlayer(1, "alice", nil), game.NewPlayer(2, "bob", nil)
			r := game.NewRoom("test", a, b, store, nil)
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			go r.Run(ctx)
			receive(t, b, "match_success")
			a.Close()
			if failure {
				receive(t, b, "error")
			}
			msg := receive(t, b, "game_over")
			raw, _ := json.Marshal(msg.Data)
			var data struct {
				WinnerID uint   `json:"winner_id"`
				Saved    bool   `json:"record_saved"`
				Reason   string `json:"reason"`
			}
			if err := json.Unmarshal(raw, &data); err != nil {
				t.Fatal(err)
			}
			if data.WinnerID != 2 || data.Saved == failure || data.Reason != "disconnect" {
				t.Fatalf("wrong result: %s", raw)
			}
			waitDone(t, r)
		})
	}
}
func TestRoomShutdown(t *testing.T) {
	store := &memoryStore{}
	m := game.NewRoomManager(store)
	a, b := game.NewPlayer(1, "a", nil), game.NewPlayer(2, "b", nil)
	r, err := m.Create(a, b)
	if err != nil {
		t.Fatal(err)
	}
	receive(t, a, "match_success")
	m.Close()
	receive(t, a, "game_over")
	waitDone(t, r)
	if _, err := m.Create(a, b); err == nil {
		t.Fatal("created room after shutdown")
	}
}
func TestSlowPlayerDoesNotBlock(t *testing.T) {
	p := game.NewPlayer(1, "slow", nil)
	for i := 0; i < 100; i++ {
		p.Emit(game.Message{Type: "ping"})
	}
	select {
	case <-p.Done:
	default:
		t.Fatal("slow player not closed")
	}
}
