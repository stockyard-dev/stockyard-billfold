package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/stockyard-dev/stockyard-billfold/internal/server"
	"github.com/stockyard-dev/stockyard-billfold/internal/store"
	"github.com/stockyard-dev/stockyard/bus"
)

var version = "dev"

func main() {
	portFlag := flag.String("port", "", "HTTP port (overrides PORT env var)")
	dataFlag := flag.String("data", "", "Data directory (overrides DATA_DIR env var)")
	flag.Parse()

	port := *portFlag
	if port == "" {
		port = os.Getenv("PORT")
	}
	if port == "" {
		port = "9700"
	}

	dataDir := *dataFlag
	if dataDir == "" {
		dataDir = os.Getenv("DATA_DIR")
	}
	if dataDir == "" {
		dataDir = "./billfold-data"
	}

	db, err := store.Open(dataDir)
	if err != nil {
		log.Fatalf("billfold: %v", err)
	}
	defer db.Close()

	// Bus lives one level up from the private data dir so every tool
	// in a bundle shares one _bus.db. Standalone use (no bundle dir)
	// is fine — the bus just becomes a private log only we can see.
	// Failures are non-fatal; billfold must boot with or without it.
	var b *bus.Bus
	if bb, berr := bus.Open(filepath.Dir(dataDir), "billfold"); berr != nil {
		log.Printf("billfold: bus disabled: %v", berr)
	} else {
		b = bb
		defer b.Close()
	}

	srv := server.New(db, server.DefaultLimits(dataDir), dataDir, b)

	fmt.Printf("\n  Billfold v%s — Self-hosted invoice generator\n", version)
	fmt.Printf("  Dashboard:  http://localhost:%s/ui\n", port)
	fmt.Printf("  API:        http://localhost:%s/api\n", port)
	fmt.Printf("  Data:       %s\n", dataDir)
	fmt.Printf("  Questions?  hello@stockyard.dev — I read every message\n\n")

	log.Printf("billfold: listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, srv))
}
