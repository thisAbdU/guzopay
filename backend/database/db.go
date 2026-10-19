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

	if err := db.AutoMigrate(&models.Driver{}, &models.Payment{}); err != nil {
		return fmt.Errorf("migrate: %w", err)
	}

	DB = db
	return Seed()
}

func Seed() error {
	var driver models.Driver
	err := DB.Where("vehicle_code = ?", "4A2Q").First(&driver).Error
	if err == gorm.ErrRecordNotFound {
		return DB.Create(&models.Driver{
			ID:               "drv-4A2Q",
			Name:             "Dawit Alemu",
			VehicleCode:      "4A2Q",
			Phone:            "0911223344",
			Route:            "Megenagna → Bole",
			WalletBalance:    0,
			ProximityEnabled: false,
			FareTiers:        models.DefaultFareTiers(),
		}).Error
	}
	if err != nil {
		return err
	}
	if len(driver.FareTiers) == 0 {
		driver.FareTiers = models.DefaultFareTiers()
		return DB.Save(&driver).Error
	}
	return nil
}
