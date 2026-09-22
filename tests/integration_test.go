package tests

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/gorilla/websocket"
	"github.com/ziwenx1973/GoArena/internal/config"
	"github.com/ziwenx1973/GoArena/internal/database"
	"github.com/ziwenx1973/GoArena/internal/game"
	"github.com/ziwenx1973/GoArena/internal/handler"
	"github.com/ziwenx1973/GoArena/internal/model"
	"github.com/ziwenx1973/GoArena/internal/repository"
	"github.com/ziwenx1973/GoArena/internal/router"
	"github.com/ziwenx1973/GoArena/internal/service"
	"golang.org/x/crypto/bcrypt"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"
)

// Requires a dedicated test database and Redis instance: matchmaking owns match:queue.
func TestIntegration(t *testing.T) {
	if os.Getenv("ARENA_INTEGRATION") != "1" {
		t.Skip("set ARENA_INTEGRATION=1 with dedicated MySQL and Redis to run")
	}
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	db, err := database.OpenMySQL(cfg)
	if err != nil {
		t.Fatal(err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		t.Fatal(err)
	}
	defer sqlDB.Close()
	redis, err := database.OpenRedis(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer redis.Close()
	users := &repository.UserRepository{DB: db}
	records := &repository.GameRepository{DB: db}
	auth := &service.AuthService{Users: users, Secret: cfg.JWTSecret}
	manager := game.NewRoomManager(records)
	mm, err := game.NewMatchMaker(redis, manager)
	if err != nil {
		t.Fatal(err)
	}
	gh := &handler.GameHandler{Auth: auth, Matches: mm}
	srv := httptest.NewServer(router.New(&handler.AuthHandler{Auth: auth}, &handler.UserHandler{Users: &service.UserService{Users: users, Games: records}}, gh))
	defer srv.Close()
	defer gh.Shutdown()
	var suffix [6]byte
	if _, err := rand.Read(suffix[:]); err != nil {
		t.Fatal(err)
	}
	prefix := "it_" + hex.EncodeToString(suffix[:])
	ids := []uint{}
	defer func() {
		if len(ids) > 0 {
			db.Where("player1_id IN ? OR player2_id IN ?", ids, ids).Delete(&model.GameRecord{})
			db.Where("id IN ?", ids).Delete(&model.User{})
		}
	}()
	request := func(method, path, token string, body any, want int) map[string]json.RawMessage {
		t.Helper()
		raw, err := json.Marshal(body)
		if err != nil {
			t.Fatal(err)
		}
		req, err := http.NewRequest(method, srv.URL+path, bytes.NewReader(raw))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if token != "" {
			req.Header.Set("Authorization", "Bearer "+token)
		}
		client := &http.Client{Timeout: 10 * time.Second}
		res, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		var out map[string]json.RawMessage
		if err := json.NewDecoder(res.Body).Decode(&out); err != nil {
			t.Fatal(err)
		}
		if res.StatusCode != want {
			t.Fatalf("%s %s status=%d want=%d body=%s", method, path, res.StatusCode, want, out)
		}
		return out
	}
	tokens := []string{}
	for _, name := range []string{prefix + "a", prefix + "b"} {
		in := map[string]string{"username": name, "password": "test-password-123"}
		out := request("POST", "/api/register", "", in, 201)
		var id uint
		if err := json.Unmarshal(out["id"], &id); err != nil {
			t.Fatal(err)
		}
		ids = append(ids, id)
		if _, ok := out["password_hash"]; ok {
			t.Fatal("hash leaked")
		}
		request("POST", "/api/register", "", in, 409)
		out = request("POST", "/api/login", "", in, 200)
		var token string
		if err := json.Unmarshal(out["token"], &token); err != nil {
			t.Fatal(err)
		}
		tokens = append(tokens, token)
		profile := request("GET", "/api/profile", token, nil, 200)
		if _, ok := profile["password_hash"]; ok {
			t.Fatal("profile hash leaked")
		}
		u, err := users.ByID(context.Background(), id)
		if err != nil || bcrypt.CompareHashAndPassword([]byte(u.PasswordHash), []byte(in["password"])) != nil {
			t.Fatal("bcrypt hash verification failed")
		}
	}
	request("POST", "/api/login", "", map[string]string{"username": prefix + "a", "password": "wrong-password"}, 401)
	request("GET", "/api/profile", "invalid", nil, 401)
	request("GET", "/api/records", "", nil, 401)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http") + "/ws?token="
	if conn, res, err := websocket.DefaultDialer.Dial(wsURL+"bad", nil); err == nil {
		conn.Close()
		t.Fatal("invalid token connected")
	} else if res == nil || res.StatusCode != 401 {
		t.Fatal("expected HTTP 401")
	}
	sockets := []*websocket.Conn{}
	for _, token := range tokens {
		c, _, err := websocket.DefaultDialer.Dial(wsURL+token, nil)
		if err != nil {
			t.Fatal(err)
		}
		sockets = append(sockets, c)
		defer c.Close()
	}
	send := func(i int, kind string, data any) {
		t.Helper()
		sockets[i].SetWriteDeadline(time.Now().Add(5 * time.Second))
		if err := sockets[i].WriteJSON(game.Message{Type: kind, Data: data}); err != nil {
			t.Fatal(err)
		}
	}
	read := func(i int, kind string) json.RawMessage {
		t.Helper()
		sockets[i].SetReadDeadline(time.Now().Add(10 * time.Second))
		var m struct {
			Type string          `json:"type"`
			Data json.RawMessage `json:"data"`
		}
		if err := sockets[i].ReadJSON(&m); err != nil {
			t.Fatal(err)
		}
		if m.Type != kind {
			t.Fatalf("player %d got %s want %s: %s", i, m.Type, kind, m.Data)
		}
		return m.Data
	}
	send(0, "ping", nil)
	read(0, "pong")
	send(0, "start_match", nil)
	read(0, "match_waiting")
	send(0, "start_match", nil)
	read(0, "error")
	send(0, "cancel_match", nil)
	read(0, "match_cancelled")
	for i := 0; i < 2; i++ {
		send(i, "start_match", nil)
		read(i, "match_waiting")
	}
	match := read(0, "match_success")
	read(1, "match_success")
	var room struct {
		RoomID string `json:"room_id"`
	}
	if err := json.Unmarshal(match, &room); err != nil {
		t.Fatal(err)
	}
	if manager.Get(room.RoomID) == nil {
		t.Fatal("room missing")
	}
	send(0, "action", map[string]string{"choice": "lizard"})
	read(0, "error")
	for round := 1; round <= 3; round++ {
		choice := "scissors"
		if round == 1 {
			choice = "rock"
		}
		send(0, "action", map[string]any{"choice": "rock", "round": round})
		if round == 1 {
			send(0, "action", map[string]any{"choice": "paper", "round": round})
			read(0, "error")
		}
		send(1, "action", map[string]any{"choice": choice, "round": round})
		read(0, "round_result")
		read(1, "round_result")
	}
	over := read(0, "game_over")
	read(1, "game_over")
	var result struct {
		Winner uint `json:"winner_id"`
		Saved  bool `json:"record_saved"`
	}
	if err := json.Unmarshal(over, &result); err != nil {
		t.Fatal(err)
	}
	if !result.Saved || result.Winner != ids[0] {
		t.Fatalf("wrong game_over: %s", over)
	}
	for _, token := range tokens {
		out := request("GET", "/api/records", token, nil, 200)
		var rows []model.GameRecord
		if err := json.Unmarshal(out["records"], &rows); err != nil {
			t.Fatal(err)
		}
		if len(rows) != 1 || rows[0].RoomID != room.RoomID || rows[0].Player1Score != 2 || rows[0].Player2Score != 0 {
			t.Fatalf("wrong history: %+v", rows)
		}
	}
	for i := 0; i < 2; i++ {
		send(i, "start_match", nil)
		read(i, "match_waiting")
	}
	read(0, "match_success")
	read(1, "match_success")
	sockets[0].Close()
	over = read(1, "game_over")
	var disconnected struct {
		Reason string `json:"reason"`
		Winner uint   `json:"winner_id"`
		Saved  bool   `json:"record_saved"`
	}
	if err := json.Unmarshal(over, &disconnected); err != nil {
		t.Fatal(err)
	}
	if disconnected.Reason != "disconnect" || disconnected.Winner != ids[1] || !disconnected.Saved {
		t.Fatalf("disconnect result: %s", over)
	}
	t.Log(fmt.Sprintf("real MySQL + Redis: register, bcrypt, login, JWT, WS, cancel, duplicate, match, draw, 2 wins, persistence, records, rematch and disconnect passed (%s)", prefix))
}
