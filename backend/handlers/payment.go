package handlers

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"guzopay/backend/database"
	"guzopay/backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

var safaricomPhone = regexp.MustCompile(`^07\d{8}$`)

// normalizePassengerPhone accepts "", 07…, or 2517… (from userScopes / Daraja).
func normalizePassengerPhone(raw string) string {
	phone := strings.TrimSpace(raw)
	phone = strings.ReplaceAll(phone, "+", "")
	phone = strings.ReplaceAll(phone, " ", "")
	if strings.HasPrefix(phone, "2517") && len(phone) == 12 {
		phone = "0" + phone[3:]
	}
	if phone == "" || safaricomPhone.MatchString(phone) {
		return phone
	}
	return phone // keep as-is; C2B webhook will overwrite
}

type PaymentHandler struct {
	Hub *Hub
}

type payRequest struct {
	VehicleCode    string  `json:"vehicle_code" binding:"required"`
	Amount         float64 `json:"amount" binding:"required"`
	FareTierName   string  `json:"fare_tier_name"`
	PassengerName  string  `json:"passenger_name"`
	PassengerPhone string  `json:"passenger_phone"`
}

func (h *PaymentHandler) GetTaxi(c *gin.Context) {
	code := strings.ToUpper(strings.TrimSpace(c.Param("code")))
	var driver models.Driver
	if err := database.DB.Where("UPPER(vehicle_code) = ?", code).First(&driver).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "taxi not found"})
		return
	}
	tiers := driver.FareTiers
	if len(tiers) == 0 {
		tiers = models.DefaultFareTiers()
	}
	c.JSON(http.StatusOK, gin.H{
		"id":                driver.ID,
		"name":              driver.Name,
		"vehicle_code":      driver.VehicleCode,
		"route":             driver.Route,
		"proximity_enabled": driver.ProximityEnabled,
		"fare_tiers":        tiers,
	})
}

func (h *PaymentHandler) NearbyTaxis(c *gin.Context) {
	var drivers []models.Driver
	database.DB.Where("proximity_enabled = ?", true).Find(&drivers)
	out := make([]gin.H, 0, len(drivers))
	for _, d := range drivers {
		tiers := d.FareTiers
		if len(tiers) == 0 {
			tiers = models.DefaultFareTiers()
		}
		out = append(out, gin.H{
			"name":         d.Name,
			"vehicle_code": d.VehicleCode,
			"route":        d.Route,
			"fare_tiers":   tiers,
		})
	}
	c.JSON(http.StatusOK, gin.H{"taxis": out})
}

func (h *PaymentHandler) Pay(c *gin.Context) {
	var req payRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	code := strings.ToUpper(strings.TrimSpace(req.VehicleCode))
	phone := normalizePassengerPhone(req.PassengerPhone)
	name := strings.TrimSpace(req.PassengerName)
	// Name/phone come from M-Pesa userScopes / C2B — never required from the UI.
	if name == "" {
		name = "M-Pesa Customer"
	}

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
	} else if len(driver.FareTiers) > 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "amount must match a configured fare tier"})
		return
	}

	txID := fmt.Sprintf("MP%s", strings.ToUpper(uuid.New().String()[:12]))
	payment := models.Payment{
		ID:             uuid.New().String(),
		VehicleCode:    driver.VehicleCode,
		PassengerName:  name,
		PassengerPhone: phone,
		Amount:         req.Amount,
		FareTierName:   tierName,
		Status:         "COMPLETED",
		MpesaTxID:      txID,
		CreatedAt:      time.Now().UTC(),
	}

	if err := database.DB.Create(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to record payment"})
		return
	}
	driver.WalletBalance += req.Amount
	if err := database.DB.Save(&driver).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to update wallet"})
		return
	}

	h.Hub.Broadcast(driver.VehicleCode, gin.H{
		"type":           "payment",
		"payment":        payment,
		"wallet_balance": driver.WalletBalance,
	})

	c.JSON(http.StatusOK, gin.H{
		"payment":        payment,
		"mpesa_tx_id":    txID,
		"driver_name":    driver.Name,
		"vehicle_code":   driver.VehicleCode,
		"wallet_balance": driver.WalletBalance,
	})
}

// GetPayment returns payment status (poll after Daraja simulate).
func (h *PaymentHandler) GetPayment(c *gin.Context) {
	id := c.Param("id")
	var payment models.Payment
	if err := database.DB.Where("id = ?", id).First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}
	var driver models.Driver
	_ = database.DB.Where("vehicle_code = ?", payment.VehicleCode).First(&driver)
	c.JSON(http.StatusOK, gin.H{
		"payment":      payment,
		"driver_name":  driver.Name,
		"vehicle_code": payment.VehicleCode,
		"mpesa_tx_id":  payment.MpesaTxID,
	})
}

// ConfirmPay finalizes a PENDING payment after payWithMpesa (transactionId) or tradePay.
func (h *PaymentHandler) ConfirmPay(c *gin.Context) {
	var req struct {
		PaymentID  string `json:"payment_id"`
		TradeNO    string `json:"tradeNO"` // M-Pesa transactionId from payWithMpesa
		ResultCode string `json:"result_code"`
	}
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var payment models.Payment
	q := database.DB
	if req.PaymentID != "" {
		q = q.Where("id = ?", req.PaymentID)
	} else if req.TradeNO != "" {
		q = q.Where("mpesa_tx_id = ?", req.TradeNO)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "payment_id or tradeNO required"})
		return
	}
	if err := q.First(&payment).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "payment not found"})
		return
	}

	code := req.ResultCode
	if code == "" {
		code = "9000"
	}
	if code != "9000" && code != "8000" && code != "6004" {
		payment.Status = "FAILED"
		_ = database.DB.Save(&payment)
		c.JSON(http.StatusOK, gin.H{"payment": payment, "status": "FAILED"})
		return
	}

	if payment.Status == "COMPLETED" {
		var driver models.Driver
		_ = database.DB.Where("vehicle_code = ?", payment.VehicleCode).First(&driver)
		c.JSON(http.StatusOK, gin.H{
			"payment":        payment,
			"mpesa_tx_id":    payment.MpesaTxID,
			"driver_name":    driver.Name,
			"vehicle_code":   payment.VehicleCode,
			"wallet_balance": driver.WalletBalance,
		})
		return
	}

	if req.TradeNO != "" {
		payment.MpesaTxID = req.TradeNO
	}
	payment.Status = "COMPLETED"
	if err := database.DB.Save(&payment).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to confirm"})
		return
	}

	var driver models.Driver
	if err := database.DB.Where("vehicle_code = ?", payment.VehicleCode).First(&driver).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "driver not found"})
		return
	}
	driver.WalletBalance += payment.Amount
	_ = database.DB.Save(&driver)

	h.Hub.Broadcast(driver.VehicleCode, gin.H{
		"type":           "payment",
		"payment":        payment,
		"wallet_balance": driver.WalletBalance,
	})

	c.JSON(http.StatusOK, gin.H{
		"payment":        payment,
		"mpesa_tx_id":    payment.MpesaTxID,
		"driver_name":    driver.Name,
		"vehicle_code":   payment.VehicleCode,
		"wallet_balance": driver.WalletBalance,
	})
}
