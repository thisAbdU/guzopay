package database

import (
	"log"

	"guzopay/backend/models"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// SeedDrivers upserts demo fleet rows (idempotent by vehicle_code).
func SeedDrivers() error {
	drivers := []models.Driver{
		{
			ID:               "drv-4A2Q",
			Name:             "Dawit Alemu",
			VehicleCode:      "4A2Q",
			Phone:            "0911223344",
			Route:            "Megenagna → Bole",
			WalletBalance:    0,
			ProximityEnabled: false,
			FareTiers:        models.DefaultFareTiers(),
		},
		{
			ID:               "drv-4421",
			Name:             "Hanna Tesfaye",
			VehicleCode:      "4421",
			Phone:            "0912334455",
			Route:            "Piassa → Mexico",
			WalletBalance:    0,
			ProximityEnabled: false,
			FareTiers: []models.FareTier{
				{Name: "Near", Amount: 7},
				{Name: "Full Route", Amount: 12},
			},
		},
		{
			ID:               "drv-A402",
			Name:             "Yonas Bekele",
			VehicleCode:      "A402",
			Phone:            "0920112233",
			Route:            "CMC → Bole Airport",
			WalletBalance:    0,
			ProximityEnabled: false,
			FareTiers: []models.FareTier{
				{Name: "Short", Amount: 10},
				{Name: "Medium", Amount: 15},
				{Name: "Full Route", Amount: 20},
			},
		},
	}

	for _, d := range drivers {
		var existing models.Driver
		err := DB.Where("vehicle_code = ?", d.VehicleCode).First(&existing).Error
		if err == gorm.ErrRecordNotFound {
			if d.ID == "" {
				d.ID = "drv-" + uuid.New().String()[:8]
			}
			if err := DB.Create(&d).Error; err != nil {
				return err
			}
			log.Printf("seed: created driver %s (%s)", d.VehicleCode, d.Name)
			continue
		}
		if err != nil {
			return err
		}
		// Refresh static profile fields; keep wallet / proximity / custom fares if already set.
		existing.Name = d.Name
		existing.Phone = d.Phone
		existing.Route = d.Route
		if len(existing.FareTiers) == 0 {
			existing.FareTiers = d.FareTiers
		}
		if err := DB.Save(&existing).Error; err != nil {
			return err
		}
	}
	return nil
}
