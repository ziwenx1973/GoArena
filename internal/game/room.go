package game

import (
	"context"
	"github.com/ziwenx1973/GoArena/internal/model"
	"log/slog"
	"time"
)

// RecordStore is the one persistence boundary used by a room and its tests.
type RecordStore interface {
	Save(context.Context, *model.GameRecord) error
}
type Room struct {
	ID        string
	Players   [2]*Player
	EventChan chan GameEvent
	Done      chan struct{}
	scores    [2]int
	choices   [2]Choice
	round     int
	store     RecordStore
	finished  func(*Room)
}

func NewRoom(id string, a, b *Player, store RecordStore, finished func(*Room)) *Room {
	return &Room{ID: id, Players: [2]*Player{a, b}, EventChan: make(chan GameEvent, 32), Done: make(chan struct{}), round: 1, store: store, finished: finished}
}
func (r *Room) Submit(e GameEvent) bool {
	select {
	case <-r.Done:
		return false
	default:
	}
	select {
	case r.EventChan <- e:
		return true
	case <-r.Done:
		return false
	default:
		return false
	}
}
func (r *Room) broadcast(t string, d any) {
	for _, p := range r.Players {
		p.Emit(Message{t, d})
	}
}
func (r *Room) Run(ctx context.Context) {
	defer close(r.Done)
	defer func() {
		if r.finished != nil {
			r.finished(r)
		}
	}()
	r.broadcast("match_success", map[string]any{"room_id": r.ID, "player1_id": r.Players[0].UserID, "player2_id": r.Players[1].UserID, "round": 1})
	timer := time.NewTimer(60 * time.Second)
	defer timer.Stop()
	for {
		select {
		case <-ctx.Done():
			r.finish(nil, "server_shutdown")
			return
		case <-r.Players[0].Done:
			id := r.Players[1].UserID
			r.finish(&id, "disconnect")
			return
		case <-r.Players[1].Done:
			id := r.Players[0].UserID
			r.finish(&id, "disconnect")
			return
		case <-timer.C:
			var winner *uint
			if r.choices[0] != "" && r.choices[1] == "" {
				id := r.Players[0].UserID
				winner = &id
			}
			if r.choices[1] != "" && r.choices[0] == "" {
				id := r.Players[1].UserID
				winner = &id
			}
			r.finish(winner, "round_timeout")
			return
		case e := <-r.EventChan:
			i := -1
			for j, p := range r.Players {
				if p.UserID == e.PlayerID {
					i = j
				}
			}
			if i < 0 {
				continue
			}
			p := r.Players[i]
			if !ValidChoice(e.Choice) {
				p.Error("invalid choice")
				continue
			}
			if e.Round != 0 && e.Round != r.round {
				p.Error("stale round")
				continue
			}
			if r.choices[i] != "" {
				p.Error("choice already submitted")
				continue
			}
			r.choices[i] = e.Choice
			if r.choices[0] == "" || r.choices[1] == "" {
				continue
			}
			result := Judge(r.choices[0], r.choices[1])
			var winner *uint
			if result != 0 {
				j := 0
				if result < 0 {
					j = 1
				}
				r.scores[j]++
				id := r.Players[j].UserID
				winner = &id
			}
			r.broadcast("round_result", map[string]any{"room_id": r.ID, "round": r.round, "player1_choice": r.choices[0], "player2_choice": r.choices[1], "winner_id": winner, "player1_score": r.scores[0], "player2_score": r.scores[1]})
			slog.Info("Round result", "room_id", r.ID, "round", r.round, "scores", r.scores)
			if r.scores[0] == 2 || r.scores[1] == 2 {
				r.finish(winner, "completed")
				return
			}
			r.round++
			r.choices = [2]Choice{}
			if !timer.Stop() {
				select {
				case <-timer.C:
				default:
				}
			}
			timer.Reset(60 * time.Second)
		}
	}
}
func (r *Room) finish(winner *uint, reason string) {
	record := model.GameRecord{RoomID: r.ID, Player1ID: r.Players[0].UserID, Player2ID: r.Players[1].UserID, WinnerID: winner, Player1Score: r.scores[0], Player2Score: r.scores[1], Reason: reason}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := r.store.Save(ctx, &record)
	if err != nil {
		slog.Error("Database error", "room_id", r.ID, "error", err)
		r.broadcast("error", map[string]string{"message": "game finished but record could not be saved"})
	}
	// Release assignments before announcing completion so immediate rematching works.
	for _, p := range r.Players {
		p.clearRoom(r)
	}
	r.broadcast("game_over", map[string]any{"room_id": r.ID, "winner_id": winner, "player1_score": r.scores[0], "player2_score": r.scores[1], "reason": reason, "record_saved": err == nil})
	slog.Info("Game over", "room_id", r.ID, "reason", reason, "record_saved", err == nil)
}
