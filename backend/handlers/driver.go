package handlers

import (
	"net/http"
	"strings"
	"time"

	"guzopay/backend/database"
	"guzopay/backend/models"

	"github.com/gin-gonic/gin"
)

type DriverHandler struct{}

// List returns all drivers (for PWA login / vehicle picker).
func (h *DriverHandler) List(c *gin.Context) {
	var drivers []models.Driver
	if err := database.DB.Order("vehicle_code asc").Find(&drivers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list drivers"})
		return
	}
	out := make([]gin.H, 0, len(drivers))
	for _, d := range drivers {
		out = append(out, gin.H{
			"id":           d.ID,
			"name":         d.Name,
			"vehicle_code": d.VehicleCode,
			"phone":        d.Phone,
			"route":        d.Route,
		})
	}
	c.JSON(http.StatusOK, gin.H{"drivers": out})
}

func (h *DriverHandler) Dashboard(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	var driver models.Driver
	if err := database.DB.Where("UPPER(vehicle_code) = ?", code).First(&driver).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "driver not found"})
		return
	}

	startOfDay := time.Now().UTC().Truncate(24 * time.Hour)
	var todayTotal float64
	var tripCount int64
	database.DB.Model(&models.Payment{}).
		Where("UPPER(vehicle_code) = ? AND status = ? AND created_at >= ?", code, "COMPLETED", startOfDay).
		Select("COALESCE(SUM(amount), 0)").Scan(&todayTotal)
	database.DB.Model(&models.Payment{}).
		Where("UPPER(vehicle_code) = ? AND status = ? AND created_at >= ?", code, "COMPLETED", startOfDay).
		Count(&tripCount)

	var payments []models.Payment
	database.DB.Where("UPPER(vehicle_code) = ?", code).
		Order("created_at desc").Limit(20).Find(&payments)

	tiers := driver.FareTiers
	if len(tiers) == 0 {
		tiers = models.DefaultFareTiers()
	}
	c.JSON(http.StatusOK, gin.H{
		"driver_name":       driver.Name,
		"vehicle_code":      driver.VehicleCode,
		"route":             driver.Route,
		"wallet_balance":    driver.WalletBalance,
		"today_total":       todayTotal,
		"trip_count":        tripCount,
		"proximity_enabled": driver.ProximityEnabled,
		"fare_tiers":        tiers,
		"transactions":      payments,
	})
}

type proximityRequest struct {
	Enabled bool `json:"enabled"`
}

func (h *DriverHandler) SetProximity(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	var req proximityRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	var driver models.Driver
	if err := database.DB.Where("UPPER(vehicle_code) = ?", code).First(&driver).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "driver not found"})
		return
	}
	driver.ProximityEnabled = req.Enabled
	if err := database.DB.Save(&driver).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update proximity"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"vehicle_code":      driver.VehicleCode,
		"proximity_enabled": driver.ProximityEnabled,
	})
}

type fareTiersRequest struct {
	FareTiers []models.FareTier `json:"fare_tiers" binding:"required"`
}

func (h *DriverHandler) SetFareTiers(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	var req fareTiersRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if len(req.FareTiers) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "at least one fare tier required"})
		return
	}
	for i, t := range req.FareTiers {
		t.Name = strings.TrimSpace(t.Name)
		if t.Name == "" || t.Amount <= 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "each tier needs a name and amount > 0"})
			return
		}
		req.FareTiers[i] = t
	}

	var driver models.Driver
	if err := database.DB.Where("UPPER(vehicle_code) = ?", code).First(&driver).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "driver not found"})
		return
	}
	driver.FareTiers = req.FareTiers
	if err := database.DB.Save(&driver).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update fare tiers"})
		return
	}
	c.JSON(http.StatusOK, gin.H{
		"vehicle_code": driver.VehicleCode,
		"fare_tiers":   driver.FareTiers,
	})
}
