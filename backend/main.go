package main

import (
	"log"
	"os"

	"guzopay/backend/database"
	"guzopay/backend/handlers"
	"guzopay/backend/mpesa"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load(".env")
	_ = godotenv.Load("backend/.env")

	if err := database.Connect(); err != nil {
		log.Fatal(err)
	}
	mpesa.InitFromEnv()

	hub := handlers.NewHub()
	pay := &handlers.PaymentHandler{Hub: hub}
	drv := &handlers.DriverHandler{}
	ws := &handlers.WSHandler{Hub: hub}
	c2b := &mpesa.C2BCallback{Broadcast: hub.Broadcast}

	r := gin.Default()
	r.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type"},
		AllowCredentials: false,
	}))

	api := r.Group("/api/v1")
	{
		api.GET("/taxi/nearby", pay.NearbyTaxis)
		api.GET("/taxi/:code", pay.GetTaxi)
		api.POST("/pay", pay.Pay)
		api.GET("/pay/:id", pay.GetPayment)
		api.POST("/pay/initiate", mpesa.InitiateMiniPay) // Mini App tradePay stub
		api.POST("/pay/daraja", mpesa.InitiateDarajaPay(hub.Broadcast))
		api.POST("/pay/confirm", pay.ConfirmPay)

		api.POST("/mpesa/c2b-validation", mpesa.C2BValidation)
		api.POST("/mpesa/c2b-callback", c2b.Handle)
		api.POST("/mpesa/b2c-result", mpesa.B2CResultCallback(hub.Broadcast))
		api.POST("/mpesa/b2c-timeout", mpesa.B2CTimeoutCallback)
		api.POST("/mpesa/register-urls", mpesa.RegisterURLsHandler)
		api.POST("/mpesa/auth", mpesa.MiniAppAuth)
		api.POST("/mpesa/notify-payment", mpesa.NotifyPayment(hub.Broadcast))

		api.GET("/drivers", drv.List)
		api.GET("/driver/:code/dashboard", drv.Dashboard)
		api.POST("/driver/:code/proximity", drv.SetProximity)
		api.PUT("/driver/:code/fare-tiers", drv.SetFareTiers)
		api.POST("/driver/:code/payout", drv.Payout)
	}
	r.GET("/ws/driver/:code", ws.DriverSocket)

	addr := ":8080"
	if p := os.Getenv("PORT"); p != "" {
		addr = ":" + p
	}
	log.Printf("GuzoPay API listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatal(err)
	}
}
