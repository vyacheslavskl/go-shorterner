package main

import (
	"log"

	"github.com/vyacheslavskl/go-shorterner/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatalf("Closed with error %v", err)
	}
}
