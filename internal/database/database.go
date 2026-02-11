package database

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"lark-robot/internal/model"
)

func Init(dbPath string, zapLogger *zap.Logger) (*gorm.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Warn), // Suppress "record not found" info logs
	})
	if err != nil {
		return nil, err
	}

	if err := db.AutoMigrate(
		&model.AutoReplyRule{},
		&model.ScheduledTask{},
		&model.MessageLog{},
		&model.Group{},
	); err != nil {
		return nil, err
	}

	zapLogger.Info("database initialized", zap.String("path", dbPath))
	return db, nil
}
