package database_test

import (
	"testing"

	"github.com/ptijjo/optiligne_back/internal/database"
)

func TestConnect_RefuseURLVide(t *testing.T) {
	_, err := database.Connect("")
	if err == nil {
		t.Fatal("attendu une erreur pour DATABASE_URL vide")
	}
}
