package id

import "github.com/nrednav/cuid2"

// New retourne un CUID2 (identifiant applicatif).
func New() string {
	return cuid2.Generate()
}
