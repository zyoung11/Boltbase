package main

import (
	// "embed"
	"log"

	"Boltbase/bolt"
)

// //go:embed web
// var webFS embed.FS

func main() {
	if err := bolt.InitDB(); err != nil {
		log.Fatalf("Failed to initialize the database\n%v", err)
	}
	defer bolt.DB.Close()
	// bolt.WebFS = webFS

	// bolt.Run("Boltbase v2.0", 5090, bolt.Routes, webFS)
	bolt.Run("Boltbase v2.0", 5090, bolt.Routes)
}
