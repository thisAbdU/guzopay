package mpesa

import (
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"guzopay/backend/database"
	"guzopay/backend/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// BroadcastFn lets the webhook push live updates without importing handlers.
type BroadcastFn func(vehicleCode string, payload any)

// C2BCallback handles Safaricom Daraja C2B confirmation callbacks.
type C2BCallback struct {
	Broadcast BroadcastFn
}

type c2bPayload struct {
	TransactionType   string  `json:"TransactionType"`
	TransID           string  `json:"TransID"`
	TransTime         string  `json:"TransTime"`
	TransAmount       float64 `json:"TransAmount"`
	BusinessShortCode string  `json:"BusinessShortCode"`
	BillRefNumber     string  `json:"BillRefNumber"` // vehicle code
	MSISDN            string  `json:"MSISDN"`
	FirstName         string  `json:"FirstName"`
	MiddleName        string  `json:"MiddleName"`
	LastName          string  `json:"LastName"`
}

func (h *C2BCallback) Handle(c *gin.Context) {
	var body c2bPayload
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"ResultCode": 1, "ResultDesc": "Invalid payload"})
		return
	}

	code := strings.ToUpper(strings.TrimSpace(body.BillRefNumber))
	if code == "" {
		c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
		return
	}

	var driver models.Driver
	if err := database.DB.Where("UPPER(vehicle_code) = ?", code).First(&driver).Error; err != nil {
		log.Printf("c2b: unknown vehicle %s", code)
		c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
		return
	}

	name := strings.TrimSpace(strings.Join([]string{body.FirstName, body.MiddleName, body.LastName}, " "))
	if name == "" {
		name = "M-Pesa Customer"
	}
	phone := FromMSISDN251(body.MSISDN)
	if !strings.HasPrefix(phone, "07") {
		phone = normalizePhone(body.MSISDN)
	}

	txID := body.TransID
	if txID == "" {
		txID = fmt.Sprintf("MP%s", strings.ToUpper(uuid.New().String()[:12]))
	}

	// Idempotent: ignore duplicate confirmation for same TransID.
	var existing models.Payment
	if err := database.DB.Where("mpesa_tx_id = ?", txID).First(&existing).Error; err == nil {
		c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
		return
	}

	// Prefer completing an existing PENDING payment for this taxi + amount.
	var payment models.Payment
	err := database.DB.
		Where("UPPER(vehicle_code) = ? AND status = ? AND amount = ?", code, "PENDING", body.TransAmount).
		Order("created_at desc").
		First(&payment).Error

	if err == nil {
		payment.Status = "COMPLETED"
		payment.MpesaTxID = txID
		payment.PassengerName = name
		payment.PassengerPhone = phone
		if err := database.DB.Save(&payment).Error; err != nil {
			log.Printf("c2b update payment: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ResultCode": 1, "ResultDesc": "Failed"})
			return
		}
	} else {
		payment = models.Payment{
			ID:             uuid.New().String(),
			VehicleCode:    driver.VehicleCode,
			PassengerName:  name,
			PassengerPhone: phone,
			Amount:         body.TransAmount,
			Status:         "COMPLETED",
			MpesaTxID:      txID,
			CreatedAt:      time.Now().UTC(),
		}
		if err := database.DB.Create(&payment).Error; err != nil {
			log.Printf("c2b create payment: %v", err)
			c.JSON(http.StatusInternalServerError, gin.H{"ResultCode": 1, "ResultDesc": "Failed"})
			return
		}
	}

	driver.WalletBalance += body.TransAmount
	_ = database.DB.Save(&driver)

	if h.Broadcast != nil {
		h.Broadcast(driver.VehicleCode, gin.H{
			"type":           "payment",
			"payment":        payment,
			"wallet_balance": driver.WalletBalance,
		})
	}

	c.JSON(http.StatusOK, gin.H{"ResultCode": 0, "ResultDesc": "Accepted"})
}

func normalizePhone(msisdn string) string {
	return FromMSISDN251(msisdn)
}
