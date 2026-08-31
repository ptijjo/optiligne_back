package ws

import (
	"encoding/json"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/ptijjo/optiligne_back/internal/guidance"
)

const writeQueueSize = 16

type incoming struct {
	Type    string  `json:"type"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	TS      int64   `json:"ts"`
	Heading float64 `json:"heading"`
}

type outgoing struct {
	Type     string  `json:"type"`
	Frac     float64 `json:"frac"`
	OffsetM  float64 `json:"offset_m"`
	NextStop string  `json:"next_stop"`
	DelayS   int     `json:"delay_s"`
	State    string  `json:"state"`
}

// Hub gère les connexions WS.
type Hub struct {
	svc       *guidance.Service
	origins   map[string]struct{}
	upgrader  websocket.Upgrader
}

func NewHub(svc *guidance.Service, allowedOrigins string) *Hub {
	allow := map[string]struct{}{}
	for _, o := range strings.Split(allowedOrigins, ",") {
		o = strings.TrimSpace(o)
		if o != "" {
			allow[o] = struct{}{}
		}
	}
	h := &Hub{svc: svc, origins: allow}
	h.upgrader = websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			origin := r.Header.Get("Origin")
			if origin == "" {
				return true
			}
			if len(allow) == 0 {
				return true
			}
			_, ok := allow[origin]
			return ok
		},
	}
	return h
}

// RegisterRoutes monte GET /ws/guidance.
func (h *Hub) RegisterRoutes(r *gin.Engine) {
	r.GET("/ws/guidance", h.serve)
}

func (h *Hub) serve(c *gin.Context) {
	sessionID := c.Query("session_id")
	if sessionID == "" || !h.svc.SessionExists(sessionID) {
		c.Status(http.StatusNotFound)
		return
	}
	conn, err := h.upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		return
	}
	conn.SetReadLimit(4096)
	out := make(chan []byte, writeQueueSize)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case msg, ok := <-out:
				if !ok {
					return
				}
				_ = conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
				if err := conn.WriteMessage(websocket.TextMessage, msg); err != nil {
					return
				}
			case <-ticker.C:
				_ = conn.WriteControl(websocket.PingMessage, nil, time.Now().Add(5*time.Second))
			}
		}
	}()
	defer func() {
		close(out)
		wg.Wait()
		_ = conn.Close()
	}()

	var lastTS int64
	for {
		_, data, err := conn.ReadMessage()
		if err != nil {
			return
		}
		var msg incoming
		if err := json.Unmarshal(data, &msg); err != nil {
			continue
		}
		if msg.Type != "position" {
			continue
		}
		if msg.Lat < -90 || msg.Lat > 90 || msg.Lon < -180 || msg.Lon > 180 {
			continue
		}
		if msg.TS > 0 && msg.TS < lastTS {
			continue
		}
		lastTS = msg.TS
		g, err := h.svc.Update(c.Request.Context(), sessionID, msg.Lat, msg.Lon)
		if err != nil {
			continue
		}
		payload, _ := json.Marshal(outgoing{
			Type: "guidance", Frac: g.Frac, OffsetM: g.OffsetM,
			NextStop: g.NextStop, DelayS: g.DelayS, State: g.State,
		})
		_ = Enqueue(out, payload)
	}
}
