package guidance_test

import (
	"testing"
	"time"

	"github.com/ptijjo/optiligne_back/internal/guidance"
)

type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func TestEndClearsActiveTrip(t *testing.T) {
	svc := guidance.NewService(nil, fixedClock{t: time.Unix(1_700_000_000, 0)}, 80, "")
	sess := &guidance.Session{
		ID:       "sess1",
		TripID:   "tripA",
		LastSeen: time.Unix(1_700_000_000, 0),
	}
	svc.PutSessionForTest(sess)

	if !svc.HasActiveTrip("tripA") {
		t.Fatal("attendu trip actif")
	}
	svc.End("sess1")
	if svc.HasActiveTrip("tripA") {
		t.Fatal("session terminée ne doit plus bloquer")
	}
	if svc.SessionExists("sess1") {
		t.Fatal("session doit être absente")
	}
}

func TestHasActiveTripPrunesIdleSession(t *testing.T) {
	start := time.Unix(1_700_000_000, 0)
	clock := &mutableClock{t: start}
	svc := guidance.NewService(nil, clock, 80, "")
	svc.PutSessionForTest(&guidance.Session{
		ID:       "old",
		TripID:   "tripB",
		LastSeen: start,
	})
	clock.t = start.Add(guidance.SessionIdleTTL + time.Minute)
	if svc.HasActiveTrip("tripB") {
		t.Fatal("session idle doit être purgée")
	}
}

type mutableClock struct{ t time.Time }

func (c *mutableClock) Now() time.Time { return c.t }
