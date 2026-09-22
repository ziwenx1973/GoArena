package repository

import (
	"context"
	"github.com/ziwenx1973/GoArena/internal/model"
	"gorm.io/gorm"
)

type GameRepository struct{ DB *gorm.DB }

func (r *GameRepository) Save(ctx context.Context, v *model.GameRecord) error {
	return r.DB.WithContext(ctx).Create(v).Error
}
func (r *GameRepository) Recent(ctx context.Context, id uint) ([]model.GameRecord, error) {
	v := make([]model.GameRecord, 0)
	err := r.DB.WithContext(ctx).Where("player1_id = ? OR player2_id = ?", id, id).Order("created_at DESC, id DESC").Limit(20).Find(&v).Error
	return v, err
}
