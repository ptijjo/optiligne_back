package main

import (
	"log"

	"github.com/ptijjo/optiligne_back/internal/importer"
)

func main() {
	if err := importer.Run(); err != nil {
		log.Fatal(err)
	}
}
