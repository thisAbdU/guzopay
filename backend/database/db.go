package database

import (
	"fmt"
	"log"
	"os"

	"guzopay/backend/models"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

var DB *gorm.DB

func Connect() error {
	dsn := os.Getenv("DATABASE_URL")
	var (
		db  *gorm.DB
		err error
	)

	if dsn == "" {
		// Zero-config local MVP; set DATABASE_URL for PostgreSQL in production.
		log.Println("DATABASE_URL unset — using SQLite ./guzopay.db")
		db, err = gorm.Open(sqlite.Open("guzopay.db"), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
	} else {
		db, err = gorm.Open(postgres.Open(dsn), &gorm.Config{
			Logger: logger.Default.LogMode(logger.Warn),
		})
	}
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}

	if err := db.AutoMigrate(&models.Driver{}, &models.Payment{}, &models.Payout{}); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	DB = db
	return SeedDrivers()
}
