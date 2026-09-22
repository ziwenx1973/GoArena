package repository

import (
	"context"
	"github.com/ziwenx1973/GoArena/internal/model"
	"gorm.io/gorm"
)

type UserRepository struct{ DB *gorm.DB }

func (r *UserRepository) Create(ctx context.Context, u *model.User) error {
	return r.DB.WithContext(ctx).Create(u).Error
}
func (r *UserRepository) ByName(ctx context.Context, name string) (*model.User, error) {
	var u model.User
	err := r.DB.WithContext(ctx).Where("username = ?", name).First(&u).Error
	return &u, err
}
func (r *UserRepository) ByID(ctx context.Context, id uint) (*model.User, error) {
	var u model.User
	err := r.DB.WithContext(ctx).First(&u, id).Error
	return &u, err
}
