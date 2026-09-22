package model

import "time"

type GameRecord struct {
	ID           uint      `json:"id" gorm:"primaryKey"`
	RoomID       string    `json:"room_id" gorm:"size:32;uniqueIndex;not null"`
	Player1ID    uint      `json:"player1_id" gorm:"index;not null"`
	Player2ID    uint      `json:"player2_id" gorm:"index;not null"`
	WinnerID     *uint     `json:"winner_id"`
	Player1Score int       `json:"player1_score"`
	Player2Score int       `json:"player2_score"`
	Reason       string    `json:"reason" gorm:"size:32;not null"`
	CreatedAt    time.Time `json:"created_at" gorm:"index"`
}
