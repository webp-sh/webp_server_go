package database

import (
	"fmt"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

type Database struct {
	*gorm.DB
}

func NewSqliteDatabaseConnection(dbPath string) (*Database, error) {
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	if err != nil {
		return nil, fmt.Errorf("failed to open Sqlite connection: %w", err)
	}
	return &Database{db}, nil
}

func (d *Database) AutoMigrate(models ...interface{}) error {
	err := d.AutoMigrate(models...)
	if err != nil {
		return fmt.Errorf("failed to auto migrate: %w", err)
	}
	return nil
}


