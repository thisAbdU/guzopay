package mpesa

import (
	"log"
	"net/http"
	"strings"

	"guzopay/backend/database"
	"guzopay/backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// C2BValidation — accept all sandbox payments (external validation optional).
func C2BValidation(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"ResultCode": 0,
		"ResultDesc": "Accepted",
	})
}

// RegisterURLsHandler POST /api/v1/mpesa/register-urls
func RegisterURLsHandler(c *gin.Context) {
	if Default == nil || !Default.Enabled() {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "mpesa client not configured"})
		return
	}
	out, err := Default.RegisterC2BURLs()
	if err != nil {
		c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "response": out})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true, "response": out})
}

type b2cResultEnvelope struct {
	Result struct {
		ResultType               int    `json:"ResultType"`
		ResultCode               int    `json:"ResultCode"`
		ResultDesc               string `json:"ResultDesc"`
		OriginatorConversationID string `json:"OriginatorConversationID"`
		ConversationID           string `json:"ConversationID"`
		TransactionID            string `json:"TransactionID"`
	} `json:"Result"`
}

// B2CResultCallback completes driver payouts.
func B2CResultCallback(broadcast BroadcastFn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body b2cResultEnvelope
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"ResultCode": 1})
			return
		}
		r := body.Result
		log.Printf("b2c result code=%d desc=%s origin=%s tx=%s",
			r.ResultCode, r.ResultDesc, r.OriginatorConversationID, r.TransactionID)

		var payout models.Payout
		q := database.DB.Where("originator_conversation_id = ?", r.OriginatorConversationID)
		if err := q.First(&payout).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
			return
		}

		if r.ResultCode == 0 {
			payout.Status = "COMPLETED"
			payout.MpesaTxID = r.TransactionID
			_ = database.DB.Save(&payout)
			if broadcast != nil {
				var driver models.Driver
				if err := database.DB.Where("vehicle_code = ?", payout.VehicleCode).First(&driver).Error; err == nil {
					broadcast(driver.VehicleCode, gin.H{
						"type":           "payout",
						"payout":         payout,
						"wallet_balance": driver.WalletBalance,
					})
				}
			}
		} else {
			// Refund ledger hold
			payout.Status = "FAILED"
			payout.FailureReason = r.ResultDesc
			_ = database.DB.Save(&payout)
			var driver models.Driver
			if err := database.DB.Where("vehicle_code = ?", payout.VehicleCode).First(&driver).Error; err == nil {
				driver.WalletBalance += payout.Amount
				_ = database.DB.Save(&driver)
				if broadcast != nil {
					broadcast(driver.VehicleCode, gin.H{
						"type":           "payout_failed",
						"payout":         payout,
						"wallet_balance": driver.WalletBalance,
					})
				}
			}
		}
		c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
	}
}

func B2CTimeoutCallback(c *gin.Context) {
	log.Println("b2c timeout callback")
	c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
}

// InitiateDarajaPay creates PENDING payment and triggers sandbox C2B simulate.
func InitiateDarajaPay(broadcast BroadcastFn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			VehicleCode    string  `json:"vehicle_code" binding:"required"`
			Amount         float64 `json:"amount" binding:"required"`
			FareTierName   string  `json:"fare_tier_name"`
			PassengerName  string  `json:"passenger_name"`
			PassengerPhone string  `json:"passenger_phone"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		code := strings.ToUpper(strings.TrimSpace(req.VehicleCode))
		var driver models.Driver
		if err := database.DB.Where("UPPER(vehicle_code) = ?", code).First(&driver).Error; err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "taxi not found"})
			return
		}
		if len(driver.FareTiers) == 0 {
			driver.FareTiers = models.DefaultFareTiers()
		}
		tierName := strings.TrimSpace(req.FareTierName)
		if tier, ok := driver.MatchFareTier(req.Amount); ok {
			if tierName == "" {
				tierName = tier.Name
			}
		} else {
			c.JSON(http.StatusBadRequest, gin.H{"error": "amount must match a configured fare tier"})
			return
		}

		phone := strings.TrimSpace(req.PassengerPhone)
		name := strings.TrimSpace(req.PassengerName)
		if name == "" {
			name = "M-Pesa Customer"
		}

		paymentID := uuid.New().String()
		// BillRef maps confirmation webhook → this taxi (and payment id suffix).
		billRef := driver.VehicleCode

		payment := models.Payment{
			ID:             paymentID,
			VehicleCode:    driver.VehicleCode,
			PassengerName:  name,
			PassengerPhone: phone,
			Amount:         req.Amount,
			FareTierName:   tierName,
			Status:         "PENDING",
			MpesaTxID:      "",
		}
		if err := database.DB.Create(&payment).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create payment"})
			return
		}

		mock := Default == nil || !Default.Enabled() || Default.ShortCode == ""
		resp := gin.H{
			"payment_id":   payment.ID,
			"vehicle_code": driver.VehicleCode,
			"driver_name":  driver.Name,
			"amount":       payment.Amount,
			"bill_ref":     billRef,
			"status":       "PENDING",
			"mock":         mock,
		}

		if !mock {
			sim, err := Default.SimulateCustomerPayment(req.Amount, phone, billRef)
			if err != nil {
				payment.Status = "FAILED"
				_ = database.DB.Save(&payment)
				c.JSON(http.StatusBadGateway, gin.H{"error": err.Error(), "payment_id": payment.ID})
				return
			}
			resp["conversation_id"] = sim.ConversationID
			resp["originator_conversation_id"] = sim.OriginatorConversationID
			resp["message"] = "Simulate accepted — waiting for C2B confirmation webhook"
		} else {
			resp["message"] = "Mock mode — use POST /pay or set MPESA_* + CALLBACK_BASE_URL"
		}

		c.JSON(http.StatusOK, resp)
	}
}
