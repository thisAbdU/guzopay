package main

import (
	"log"
	"os"

	"guzopay/backend/database"
	"guzopay/backend/handlers"
	"guzopay/backend/mpesa"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	if err := database.Connect(); err != nil {
		log.Fatal(err)
	}

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
		api.POST("/pay/initiate", mpesa.InitiateMiniPay)
		api.POST("/pay/confirm", pay.ConfirmPay)
		api.POST("/mpesa/c2b-callback", c2b.Handle)
		api.POST("/mpesa/auth", mpesa.MiniAppAuth)
		api.POST("/mpesa/notify-payment", mpesa.NotifyPayment(hub.Broadcast))
		api.GET("/driver/:code/dashboard", drv.Dashboard)
		api.POST("/driver/:code/proximity", drv.SetProximity)
		api.PUT("/driver/:code/fare-tiers", drv.SetFareTiers)
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
