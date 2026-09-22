package main

import (
	"context"
	"errors"
	"github.com/ziwenx1973/GoArena/internal/config"
	"github.com/ziwenx1973/GoArena/internal/database"
	"github.com/ziwenx1973/GoArena/internal/game"
	"github.com/ziwenx1973/GoArena/internal/handler"
	"github.com/ziwenx1973/GoArena/internal/repository"
	"github.com/ziwenx1973/GoArena/internal/router"
	"github.com/ziwenx1973/GoArena/internal/service"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"
)

func main() {
	if err := run(); err != nil {
		slog.Error("Server stopped", "error", err)
		os.Exit(1)
	}
}
func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	db, err := database.OpenMySQL(cfg)
	if err != nil {
		return err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}
	defer sqlDB.Close()
	slog.Info("MySQL connected")
	redis, err := database.OpenRedis(cfg)
	if err != nil {
		return err
	}
	defer redis.Close()
	slog.Info("Redis connected")
	users := &repository.UserRepository{DB: db}
	records := &repository.GameRepository{DB: db}
	auth := &service.AuthService{Users: users, Secret: cfg.JWTSecret}
	rooms := game.NewRoomManager(records)
	matches, err := game.NewMatchMaker(redis, rooms)
	if err != nil {
		rooms.Close()
		return err
	}
	games := &handler.GameHandler{Auth: auth, Matches: matches}
	defer games.Shutdown()
	server := &http.Server{Addr: ":" + cfg.Port, Handler: router.New(&handler.AuthHandler{Auth: auth}, &handler.UserHandler{Users: &service.UserService{Users: users, Games: records}}, games), ReadHeaderTimeout: 5 * time.Second, ReadTimeout: 10 * time.Second, WriteTimeout: 15 * time.Second, IdleTimeout: 60 * time.Second}
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	errs := make(chan error, 1)
	go func() { slog.Info("Server started", "address", server.Addr); errs <- server.ListenAndServe() }()
	select {
	case err = <-errs:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdown); err != nil {
		_ = server.Close()
		return err
	}
	return nil
}
