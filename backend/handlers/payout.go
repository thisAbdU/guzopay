package handlers

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"guzopay/backend/database"
	"guzopay/backend/models"
	"guzopay/backend/mpesa"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type payoutRequest struct {
	Amount float64 `json:"amount" binding:"required"`
}

func (h *DriverHandler) Payout(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	var req payoutRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if req.Amount <= 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must be > 0"})
		return
	}

	var driver models.Driver
	if err := database.DB.Where("UPPER(vehicle_code) = ?", code).First(&driver).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "driver not found"})
		return
	}
	if driver.WalletBalance < req.Amount {
		c.JSON(http.StatusBadRequest, gin.H{"error": "insufficient ledger balance"})
		return
	}
	if driver.Phone == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "driver phone not set"})
		return
	}

	originatorID := fmt.Sprintf("GuzoPay-%s", uuid.New().String())
	payout := models.Payout{
		ID:                       uuid.New().String(),
		VehicleCode:              driver.VehicleCode,
		DriverPhone:              driver.Phone,
		Amount:                   req.Amount,
		Status:                   "PENDING",
		OriginatorConversationID: originatorID,
		CreatedAt:                time.Now().UTC(),
	}

	// Hold funds on ledger until B2C result.
	driver.WalletBalance -= req.Amount
	if err := database.DB.Save(&driver).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to hold balance"})
		return
	}
	if err := database.DB.Create(&payout).Error; err != nil {
		driver.WalletBalance += req.Amount
		_ = database.DB.Save(&driver)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record payout"})
		return
	}

	if mpesa.Default == nil || !mpesa.Default.Enabled() {
		// Mock: complete instantly for local demo without Daraja.
		payout.Status = "COMPLETED"
		payout.MpesaTxID = "MOCK-" + originatorID[:8]
		_ = database.DB.Save(&payout)
		c.JSON(http.StatusOK, gin.H{
			"payout":         payout,
			"wallet_balance": driver.WalletBalance,
			"mock":           true,
		})
		return
	}

	res, err := mpesa.Default.B2CPayOut(originatorID, req.Amount, driver.Phone, "GuzoPay driver payout")
	if err != nil {
		payout.Status = "FAILED"
		payout.FailureReason = err.Error()
		_ = database.DB.Save(&payout)
		driver.WalletBalance += req.Amount
		_ = database.DB.Save(&driver)
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "payout": payout})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"payout":                       payout,
		"wallet_balance":               driver.WalletBalance,
		"conversation_id":              res.ConversationID,
		"originator_conversation_id":   res.OriginatorConversationID,
		"message":                      "B2C accepted — waiting for result callback",
	})
}
