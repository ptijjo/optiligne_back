package main

import (
	"log"

	"github.com/ptijjo/optiligne_back/internal/seed"
)

func main() {
	if err := seed.Run(); err != nil {
		log.Fatal(err)
	}
}
