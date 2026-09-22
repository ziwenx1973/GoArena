-- Reference schema; startup AutoMigrate also creates these tables.
-- Run inside your chosen database (default: go_arena).
CREATE TABLE IF NOT EXISTS users (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  username VARCHAR(32) NOT NULL,
  password_hash VARCHAR(255) NOT NULL,
  created_at DATETIME(3),
  updated_at DATETIME(3),
  UNIQUE KEY idx_users_username (username)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
CREATE TABLE IF NOT EXISTS game_records (
  id BIGINT UNSIGNED AUTO_INCREMENT PRIMARY KEY,
  room_id VARCHAR(32) NOT NULL,
  player1_id BIGINT UNSIGNED NOT NULL,
  player2_id BIGINT UNSIGNED NOT NULL,
  winner_id BIGINT UNSIGNED NULL,
  player1_score BIGINT NOT NULL DEFAULT 0,
  player2_score BIGINT NOT NULL DEFAULT 0,
  reason VARCHAR(32) NOT NULL,
  created_at DATETIME(3),
  UNIQUE KEY idx_game_records_room_id (room_id),
  KEY idx_game_records_player1_id (player1_id),
  KEY idx_game_records_player2_id (player2_id),
  KEY idx_game_records_created_at (created_at)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4;
