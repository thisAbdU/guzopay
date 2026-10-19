package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type FareTier struct {
	Name   string  `json:"name"`   // e.g. "Near", "Medium", "Full Route"
	Amount float64 `json:"amount"` // e.g. 5.0, 10.0, 15.0
}

type Driver struct {
	ID               string     `json:"id" gorm:"primaryKey"`
	Name             string     `json:"name"`
	VehicleCode      string     `json:"vehicle_code" gorm:"uniqueIndex;size:8"`
	Phone            string     `json:"phone"`
	Route            string     `json:"route"`
	WalletBalance    float64    `json:"wallet_balance"`
	ProximityEnabled bool       `json:"proximity_enabled" gorm:"default:false"`
	FareTiers        []FareTier `json:"fare_tiers" gorm:"serializer:json"`
}

type Payment struct {
	ID             string    `json:"id" gorm:"type:uuid;primaryKey"`
	VehicleCode    string    `json:"vehicle_code" gorm:"index"`
	PassengerName  string    `json:"passenger_name"`
	PassengerPhone string    `json:"passenger_phone"` // Safaricom ET: starts with 07
	Amount         float64   `json:"amount"`
	FareTierName   string    `json:"fare_tier_name"`
	Status         string    `json:"status"` // COMPLETED | PENDING
	MpesaTxID      string    `json:"mpesa_tx_id"`
	CreatedAt      time.Time `json:"created_at"`
}

func (p *Payment) BeforeCreate(tx *gorm.DB) error {
	if p.ID == "" {
		p.ID = uuid.New().String()
	}
	return nil
}

func DefaultFareTiers() []FareTier {
	return []FareTier{
		{Name: "Near", Amount: 5},
		{Name: "Medium", Amount: 10},
		{Name: "Full Route", Amount: 15},
	}
}

func (d *Driver) MatchFareTier(amount float64) (FareTier, bool) {
	for _, t := range d.FareTiers {
		if t.Amount == amount {
			return t, true
		}
	}
	return FareTier{}, false
}
