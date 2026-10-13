package guidance_test

import (
	"testing"
	"time"

	"github.com/ptijjo/optiligne_back/internal/guidance"
)

func TestServiceMidnightParis_EteDecalage2h(t *testing.T) {
	midnight, err := guidance.ServiceMidnightParis("2026-07-15")
	if err != nil {
		t.Fatal(err)
	}
	if midnight.Location().String() != "Europe/Paris" {
		t.Fatalf("loc=%s", midnight.Location())
	}
	// 08:00 Paris en été = 06:00 UTC — sans Paris, retard/avance serait faussé de 2 h.
	arrival := midnight.Add(8 * time.Hour)
	utcHour, _, _ := arrival.UTC().Clock()
	if utcHour != 6 {
		t.Fatalf("08:00 Paris → UTC hour=%d (attendu 6)", utcHour)
	}
	utcMidnight := time.Date(2026, 7, 15, 0, 0, 0, 0, time.UTC)
	wrong := utcMidnight.Add(8 * time.Hour)
	if arrival.Equal(wrong) {
		t.Fatal("Paris et UTC ne doivent pas coïncider en été")
	}
	if wrong.Sub(arrival) != 2*time.Hour {
		t.Fatalf("décalage été attendu 2h, got %v", wrong.Sub(arrival))
	}
}

func TestServiceMidnightParis_HiverDecalage1h(t *testing.T) {
	midnight, err := guidance.ServiceMidnightParis("2026-01-15")
	if err != nil {
		t.Fatal(err)
	}
	arrival := midnight.Add(8 * time.Hour)
	utcHour, _, _ := arrival.UTC().Clock()
	if utcHour != 7 {
		t.Fatalf("08:00 Paris hiver → UTC hour=%d (attendu 7)", utcHour)
	}
}

func TestServiceMidnightParis_DateInvalide(t *testing.T) {
	if _, err := guidance.ServiceMidnightParis("pas-une-date"); err == nil {
		t.Fatal("attendu erreur")
	}
}
