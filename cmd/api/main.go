package main

import (
	"log"

	"github.com/ptijjo/optiligne_back/internal/app"
)

func main() {
	if err := app.Run(); err != nil {
		log.Fatal(err)
	}
}
