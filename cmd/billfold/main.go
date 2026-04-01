package main

import (
	"log"
	"os"

	"github.com/stockyard-dev/stockyard-billfold/internal/license"
	"github.com/stockyard-dev/stockyard-billfold/internal/server"
	"github.com/stockyard-dev/stockyard-billfold/internal/store"
)

func main() {
	port := getEnv("PORT", "9270")
	dataDir := getEnv("DATA_DIR", "./data")
	licenseKey := os.Getenv("BILLFOLD_LICENSE_KEY")

	tier := "free"
	if licenseKey != "" {
		if license.Validate(licenseKey) {
			tier = "pro"
			log.Println("License valid — Pro tier active")
		} else {
			log.Println("Warning: invalid license key, running as free tier")
		}
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("store: %v", err)
	}
	defer db.Close()

	srv := server.New(db, tier)
	log.Printf("Stockyard Billfold listening on :%s (tier: %s)", port, tier)
	log.Fatal(srv.ListenAndServe(":" + port))
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
