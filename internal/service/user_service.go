package service

import (
	"context"
	"github.com/ziwenx1973/GoArena/internal/model"
	"github.com/ziwenx1973/GoArena/internal/repository"
)

type UserService struct {
	Users *repository.UserRepository
	Games *repository.GameRepository
}

func (s *UserService) Profile(ctx context.Context, id uint) (*model.User, error) {
	return s.Users.ByID(ctx, id)
}
func (s *UserService) Records(ctx context.Context, id uint) ([]model.GameRecord, error) {
	return s.Games.Recent(ctx, id)
}
