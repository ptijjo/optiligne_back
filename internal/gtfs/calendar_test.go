package gtfs_test

import (
	"testing"
	"time"

	"github.com/ptijjo/optiligne_back/internal/gtfs"
)

func TestServiceActive_LundiDansPlage(t *testing.T) {
	cals := []gtfs.Calendar{{
		ServiceID: "SVC1",
		Monday:    true,
		StartDate: "20260101",
		EndDate:   "20261231",
	}}
	day := time.Date(2026, 1, 5, 0, 0, 0, 0, time.UTC) // lundi
	if !gtfs.ServiceActive(cals, nil, "SVC1", day) {
		t.Fatal("lundi devrait être actif")
	}
}

func TestServiceActive_ExceptionRemoved(t *testing.T) {
	cals := []gtfs.Calendar{{
		ServiceID: "SVC1",
		Saturday:  true,
		StartDate: "20260101",
		EndDate:   "20261231",
	}}
	dates := []gtfs.CalendarDate{{ServiceID: "SVC1", Date: "20260103", ExceptionType: 2}}
	day := time.Date(2026, 1, 3, 0, 0, 0, 0, time.UTC) // samedi
	if gtfs.ServiceActive(cals, dates, "SVC1", day) {
		t.Fatal("exception 2 doit retirer le service")
	}
}
