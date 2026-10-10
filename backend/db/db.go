package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func OpenPostgres(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	// Enable pg_trgm extension for typo-tolerant fuzzy matching across videos & users
	_ = db.Exec("CREATE EXTENSION IF NOT EXISTS pg_trgm;").Error
	return db, nil
}

func NewRepository(database *gorm.DB) *Repository {
	return &Repository{orm: database}
}

type Repository struct {
	orm *gorm.DB
}
