package ws_test

import (
	"testing"

	"github.com/ptijjo/optiligne_back/internal/ws"
)

func TestEnqueue_DropSiPleine(t *testing.T) {
	ch := make(chan []byte, 1)
	if !ws.Enqueue(ch, []byte("a")) {
		t.Fatal("premier enqueue")
	}
	if ws.Enqueue(ch, []byte("b")) {
		t.Fatal("queue pleine doit drop")
	}
}
