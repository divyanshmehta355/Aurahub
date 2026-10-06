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
	
	// Add columns if they don't exist
	db.Exec(`ALTER TABLE users ADD COLUMN IF NOT EXISTS show_adult_content BOOLEAN DEFAULT FALSE;`)
	db.Exec(`ALTER TABLE videos ADD COLUMN IF NOT EXISTS is_adult BOOLEAN DEFAULT FALSE;`)
	
	return db, nil
}

func NewRepository(database *gorm.DB) *Repository {
	return &Repository{orm: database}
}

type Repository struct {
	orm *gorm.DB
}
