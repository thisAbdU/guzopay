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
// Stub-ready for github.com/Safaricom-Ethiopia-PLC/mpesa-go-sdk integration.
type C2BCallback struct {
	Broadcast BroadcastFn
}

// Daraja-style confirmation payload (subset used by the MVP).
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
	phone := normalizePhone(body.MSISDN)

	payment := models.Payment{
		ID:             uuid.New().String(),
		VehicleCode:    driver.VehicleCode,
		PassengerName:  name,
		PassengerPhone: phone,
		Amount:         body.TransAmount,
		Status:         "COMPLETED",
		MpesaTxID:      body.TransID,
		CreatedAt:      time.Now().UTC(),
	}
	if payment.MpesaTxID == "" {
		payment.MpesaTxID = fmt.Sprintf("MP%s", strings.ToUpper(uuid.New().String()[:12]))
	}

	if err := database.DB.Create(&payment).Error; err != nil {
		log.Printf("c2b create payment: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"ResultCode": 1, "ResultDesc": "Failed"})
		return
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
	msisdn = strings.TrimSpace(msisdn)
	if strings.HasPrefix(msisdn, "2517") && len(msisdn) == 12 {
		return "0" + msisdn[3:]
	}
	if strings.HasPrefix(msisdn, "07") {
		return msisdn
	}
	return msisdn
}
