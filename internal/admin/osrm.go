package admin

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

// Router calcule un tracé routier (OSRM /route) ou colle une trace (/match).
type Router interface {
	Route(ctx context.Context, lngLats [][2]float64) ([][]float64, error)
	Match(ctx context.Context, lngLats [][2]float64) ([][]float64, error)
}

// OSRMClient appelle l'API HTTP OSRM.
type OSRMClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

func NewOSRM(baseURL string) *OSRMClient {
	return &OSRMClient{
		BaseURL: strings.TrimRight(baseURL, "/"),
		HTTPClient: &http.Client{Timeout: 20 * time.Second},
	}
}

func (o *OSRMClient) Route(ctx context.Context, lngLats [][2]float64) ([][]float64, error) {
	if len(lngLats) < 2 {
		return nil, ErrShapeTooShort
	}
	parts := make([]string, 0, len(lngLats))
	for _, p := range lngLats {
		parts = append(parts, fmt.Sprintf("%f,%f", p[0], p[1]))
	}
	u, err := url.Parse(o.BaseURL + "/route/v1/driving/" + strings.Join(parts, ";"))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("overview", "full")
	q.Set("geometries", "geojson")
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := o.HTTPClient.Do(req)
	if err != nil {
		return nil, ErrOSRM
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, ErrOSRM
	}
	var parsed struct {
		Code   string `json:"code"`
		Routes []struct {
			Geometry struct {
				Coordinates [][]float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"routes"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.Code != "Ok" || len(parsed.Routes) == 0 {
		return nil, ErrOSRM
	}
	coords := parsed.Routes[0].Geometry.Coordinates
	if len(coords) < 2 {
		return nil, ErrShapeTooShort
	}
	return coords, nil
}

const (
	matchMaxPoints = 80
	matchChunkSize = 10 // démo public OSRM : TooBig au-delà de ~10–12 points
	matchRadiusM   = 25
)

func downsampleLngLats(coords [][2]float64, max int) [][2]float64 {
	if max < 2 || len(coords) <= max {
		return coords
	}
	last := len(coords) - 1
	out := make([][2]float64, 0, max)
	for i := 0; i < max-1; i++ {
		idx := i * last / (max - 1)
		out = append(out, coords[idx])
	}
	out = append(out, coords[last])
	return out
}

func (o *OSRMClient) Match(ctx context.Context, lngLats [][2]float64) ([][]float64, error) {
	if len(lngLats) < 2 {
		return nil, ErrShapeTooShort
	}
	lngLats = downsampleLngLats(lngLats, matchMaxPoints)

	// 1. Un seul appel si la trace tient dans la limite démo / instance locale.
	if len(lngLats) <= matchChunkSize {
		return o.matchOnce(ctx, lngLats)
	}

	// 2. Sinon fenêtres chevauchantes (dernier point = premier du suivant).
	var out [][]float64
	for start := 0; start < len(lngLats)-1; {
		end := start + matchChunkSize
		if end > len(lngLats) {
			end = len(lngLats)
		}
		chunk := lngLats[start:end]
		seg, err := o.matchOnce(ctx, chunk)
		if err != nil {
			return nil, err
		}
		out = appendMatched(out, seg)
		if end == len(lngLats) {
			break
		}
		start = end - 1 // chevauchement 1 point
	}
	if len(out) < 2 {
		return nil, ErrShapeTooShort
	}
	return out, nil
}

func appendMatched(dst, seg [][]float64) [][]float64 {
	if len(seg) == 0 {
		return dst
	}
	if len(dst) > 0 {
		last := dst[len(dst)-1]
		first := seg[0]
		if last[0] == first[0] && last[1] == first[1] {
			seg = seg[1:]
		}
	}
	return append(dst, seg...)
}

func (o *OSRMClient) matchOnce(ctx context.Context, lngLats [][2]float64) ([][]float64, error) {
	parts := make([]string, 0, len(lngLats))
	radii := make([]string, 0, len(lngLats))
	for _, p := range lngLats {
		parts = append(parts, fmt.Sprintf("%f,%f", p[0], p[1]))
		radii = append(radii, fmt.Sprintf("%d", matchRadiusM))
	}
	u, err := url.Parse(o.BaseURL + "/match/v1/driving/" + strings.Join(parts, ";"))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("overview", "full")
	q.Set("geometries", "geojson")
	q.Set("gaps", "ignore")
	q.Set("radiuses", strings.Join(radii, ";"))
	u.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	res, err := o.HTTPClient.Do(req)
	if err != nil {
		return nil, ErrOSRM
	}
	defer res.Body.Close()
	raw, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, ErrOSRM
	}
	var parsed struct {
		Code      string `json:"code"`
		Matchings []struct {
			Geometry struct {
				Coordinates [][]float64 `json:"coordinates"`
			} `json:"geometry"`
		} `json:"matchings"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil || parsed.Code != "Ok" || len(parsed.Matchings) == 0 {
		return nil, ErrOSRM
	}
	var coords [][]float64
	for _, m := range parsed.Matchings {
		coords = appendMatched(coords, m.Geometry.Coordinates)
	}
	if len(coords) < 2 {
		return nil, ErrShapeTooShort
	}
	return coords, nil
}
