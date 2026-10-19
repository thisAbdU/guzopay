package mpesa

import (
	"net/http"
	"os"
	"strings"
	"time"

	"guzopay/backend/database"
	"guzopay/backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// MiniAppAuth exchanges a mini-app authCode for passenger profile.
// Stub until Safaricom applyToken OpenAPI credentials are configured.
// Docs: my.getAuthCode → POST /v2/authorizations/applyToken
func MiniAppAuth(c *gin.Context) {
	var req struct {
		AuthCode string `json:"auth_code" binding:"required"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Real path: call Safaricom OpenAPI applyToken with AUTH_CLIENT_ID / APP_ID.
	if os.Getenv("MPESA_AUTH_CLIENT_ID") != "" && os.Getenv("MPESA_APP_ID") != "" {
		// ⚠️ verify: wire applyToken HTTP call when portal credentials are issued.
		c.JSON(http.StatusOK, gin.H{
			"status":  "pending_openapi",
			"message": "applyToken credentials present — implement OpenAPI exchange",
			"auth_code": req.AuthCode,
		})
		return
	}

	// Hackathon / browser-dev stub profile from auth code fingerprint.
	suffix := req.AuthCode
	if len(suffix) > 4 {
		suffix = suffix[len(suffix)-4:]
	}
	c.JSON(http.StatusOK, gin.H{
		"passenger_name":  "M-Pesa User",
		"passenger_phone": "0712" + padPhone(suffix),
		"customer_id":     "cust-" + suffix,
		"mock":            true,
	})
}

func padPhone(s string) string {
	digits := make([]rune, 0, 6)
	for _, r := range s {
		if r >= '0' && r <= '9' {
			digits = append(digits, r)
		}
	}
	for len(digits) < 6 {
		digits = append(digits, '0')
	}
	return string(digits[:6])
}

// NotifyPayment receives wallet payment status callbacks (OpenAPI notifyPayment).
func NotifyPayment(broadcast BroadcastFn) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			PaymentID      string  `json:"payment_id"`
			TradeNO        string  `json:"tradeNO"`
			AcquirementID  string  `json:"acquirementId"`
			PaymentStatus  string  `json:"paymentStatus"`
			TransAmount    float64 `json:"transAmount"`
			BillRefNumber  string  `json:"billRefNumber"`
			CustomerName   string  `json:"customerName"`
			MSISDN         string  `json:"msisdn"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"result": "FAIL"})
			return
		}

		status := strings.ToUpper(body.PaymentStatus)
		if status == "" {
			status = "SUCCESS"
		}

		var payment models.Payment
		q := database.DB
		if body.PaymentID != "" {
			q = q.Where("id = ?", body.PaymentID)
		} else if body.TradeNO != "" {
			q = q.Where("mpesa_tx_id = ?", body.TradeNO)
		} else {
			c.JSON(http.StatusOK, gin.H{"result": "SUCCESS"})
			return
		}
		if err := q.First(&payment).Error; err != nil {
			c.JSON(http.StatusOK, gin.H{"result": "SUCCESS"})
			return
		}

		if status == "SUCCESS" || status == "COMPLETED" {
			if payment.Status != "COMPLETED" {
				payment.Status = "COMPLETED"
				_ = database.DB.Save(&payment)
				var driver models.Driver
				if err := database.DB.Where("vehicle_code = ?", payment.VehicleCode).First(&driver).Error; err == nil {
					driver.WalletBalance += payment.Amount
					_ = database.DB.Save(&driver)
					if broadcast != nil {
						broadcast(driver.VehicleCode, gin.H{
							"type":           "payment",
							"payment":        payment,
							"wallet_balance": driver.WalletBalance,
						})
					}
				}
			}
		} else if status == "FAIL" || status == "FAILED" {
			payment.Status = "FAILED"
			_ = database.DB.Save(&payment)
		}

		c.JSON(http.StatusOK, gin.H{"result": "SUCCESS"})
	}
}

// InitiateMiniPay creates a PENDING payment and returns a tradeNO for my.tradePay.
func InitiateMiniPay(c *gin.Context) {
	var req struct {
		VehicleCode    string  `json:"vehicle_code" binding:"required"`
		Amount         float64 `json:"amount" binding:"required"`
		FareTierName   string  `json:"fare_tier_name"`
		PassengerName  string  `json:"passenger_name"`
		PassengerPhone string  `json:"passenger_phone"`
		AuthCode       string  `json:"auth_code"`
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

	// PENDING until payWithMpesa success or C2B confirmation webhook.
	payment := models.Payment{
		ID:             uuid.New().String(),
		VehicleCode:    driver.VehicleCode,
		PassengerName:  name,
		PassengerPhone: phone,
		Amount:         req.Amount,
		FareTierName:   tierName,
		Status:         "PENDING",
		MpesaTxID:      "",
		CreatedAt:      time.Now().UTC(),
	}
	if err := database.DB.Create(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create order"})
		return
	}

	businessID := os.Getenv("MPESA_SHORTCODE")
	c.JSON(http.StatusOK, gin.H{
		"payment_id":     payment.ID,
		"bill_reference": driver.VehicleCode, // pass to payWithMpesa billReference
		"business_id":    businessID,
		"amount":         payment.Amount,
		"vehicle_code":   driver.VehicleCode,
		"driver_name":    driver.Name,
		"status":         "PENDING",
	})
}
