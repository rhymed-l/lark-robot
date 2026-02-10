package database

import (
	"os"
	"path/filepath"

	"go.uber.org/zap"
	"github.com/glebarez/sqlite"
	"gorm.io/gorm"

	"lark-robot/internal/model"
)

func Init(dbPath string, logger *zap.Logger) (*gorm.DB, error) {
	dir := filepath.Dir(dbPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}

	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
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

	logger.Info("database initialized", zap.String("path", dbPath))
	return db, nil
}
