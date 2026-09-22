package database

import (
	"context"
	driver "github.com/go-sql-driver/mysql"
	"github.com/ziwenx1973/GoArena/internal/config"
	"github.com/ziwenx1973/GoArena/internal/model"
	"gorm.io/driver/mysql"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
	"net"
	"time"
)

func OpenMySQL(c config.Config) (*gorm.DB, error) {
	d := driver.NewConfig()
	d.User = c.MySQLUser
	d.Passwd = c.MySQLPassword
	d.Net = "tcp"
	d.Addr = net.JoinHostPort(c.MySQLHost, c.MySQLPort)
	d.DBName = c.MySQLDatabase
	d.ParseTime = true
	d.Loc = time.UTC
	d.Timeout = 5 * time.Second
	d.ReadTimeout = 5 * time.Second
	d.WriteTimeout = 5 * time.Second
	d.Params = map[string]string{"charset": "utf8mb4"}
	db, err := gorm.Open(mysql.Open(d.FormatDSN()), &gorm.Config{TranslateError: true, Logger: logger.Default.LogMode(logger.Silent)})
	if err != nil {
		return nil, err
	}
	sqlDB, err := db.DB()
	if err != nil {
		return nil, err
	}
	sqlDB.SetMaxOpenConns(10)
	sqlDB.SetMaxIdleConns(5)
	sqlDB.SetConnMaxLifetime(time.Hour)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err = db.WithContext(ctx).AutoMigrate(&model.User{}, &model.GameRecord{}); err != nil {
		sqlDB.Close()
		return nil, err
	}
	return db, nil
}
