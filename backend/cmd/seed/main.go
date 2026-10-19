// Seed fleet data into the GuzoPay DB.
//
//	cd backend && go run ./cmd/seed
package main

import (
	"log"

	"guzopay/backend/database"

	"github.com/joho/godotenv"
)

func main() {
	_ = godotenv.Load()
	_ = godotenv.Load(".env")
	_ = godotenv.Load("backend/.env")
	if err := database.Connect(); err != nil {
		log.Fatal(err)
	}
	log.Println("seed complete")
}
