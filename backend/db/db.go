package db

import (
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func OpenPostgres(dsn string) (*gorm.DB, error) {
	return gorm.Open(postgres.Open(dsn), &gorm.Config{})
}

func NewRepository(database *gorm.DB) *Repository {
	return &Repository{orm: database}
}

type Repository struct {
	orm *gorm.DB
}
