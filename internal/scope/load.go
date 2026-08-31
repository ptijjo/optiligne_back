package scope

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ptijjo/optiligne_back/internal/gtfs"
)

const (
	RouteTypeSchoolBus     = 712
	RouteTypeSchoolPublic  = 713
)

// Perimeter est le graphe transporteur / dépôt / routes GTFS.
type Perimeter struct {
	Operators []Operator
	Depots    []Depot
	Assigns   []Assignment
}

type Operator struct {
	Code, Name string
}

type Depot struct {
	Code, OperatorCode, Name string
}

type Assignment struct {
	OperatorCode, DepotCode, Ligne, RouteID string
}

// Load lit data/perimetres et résout `ligne` contre les routes GTFS.
func Load(dir string, routes []gtfs.Route) (*Perimeter, error) {
	ops, err := readOperators(filepath.Join(dir, "operators.csv"))
	if err != nil {
		return nil, err
	}
	deps, err := readDepots(filepath.Join(dir, "depots.csv"))
	if err != nil {
		return nil, err
	}
	raw, err := readAssignments(filepath.Join(dir, "assignments.csv"))
	if err != nil {
		return nil, err
	}
	index := routeIndex(routes)
	var assigns []Assignment
	for _, a := range raw {
		id, ok := index[normalize(a.Ligne)]
		if !ok {
			return nil, fmt.Errorf("scope: ligne %q absente du GTFS", a.Ligne)
		}
		a.RouteID = id
		assigns = append(assigns, a)
	}
	return &Perimeter{Operators: ops, Depots: deps, Assigns: assigns}, nil
}

// RouteIDs retourne les route_id GTFS du couple operator+depot.
func (p *Perimeter) RouteIDs(operatorCode, depotCode string) []string {
	var out []string
	for _, a := range p.Assigns {
		if a.OperatorCode == operatorCode && a.DepotCode == depotCode {
			out = append(out, a.RouteID)
		}
	}
	return out
}

// ContainsRoute indique si la route est dans le périmètre.
func (p *Perimeter) ContainsRoute(operatorCode, depotCode, routeID string) bool {
	for _, a := range p.Assigns {
		if a.OperatorCode == operatorCode && a.DepotCode == depotCode && a.RouteID == routeID {
			return true
		}
	}
	return false
}

// IsSchoolRouteType est le filtre prise de poste (712/713).
func IsSchoolRouteType(routeType int) bool {
	return routeType == RouteTypeSchoolBus || routeType == RouteTypeSchoolPublic
}

func routeIndex(routes []gtfs.Route) map[string]string {
	m := make(map[string]string, len(routes)*2)
	for _, r := range routes {
		m[normalize(r.RouteID)] = r.RouteID
		if r.ShortName != "" {
			m[normalize(r.ShortName)] = r.RouteID
		}
	}
	return m
}

func normalize(s string) string {
	return strings.ToUpper(strings.TrimSpace(s))
}

func readOperators(path string) ([]Operator, error) {
	var out []Operator
	err := eachCSV(path, func(h, row []string) error {
		out = append(out, Operator{Code: csvCol(h, row, "code"), Name: csvCol(h, row, "name")})
		return nil
	})
	return out, err
}

func readDepots(path string) ([]Depot, error) {
	var out []Depot
	err := eachCSV(path, func(h, row []string) error {
		out = append(out, Depot{
			Code:         csvCol(h, row, "code"),
			OperatorCode: csvCol(h, row, "operator_code"),
			Name:         csvCol(h, row, "name"),
		})
		return nil
	})
	return out, err
}

func readAssignments(path string) ([]Assignment, error) {
	var out []Assignment
	err := eachCSV(path, func(h, row []string) error {
		out = append(out, Assignment{
			OperatorCode: csvCol(h, row, "operator_code"),
			DepotCode:    csvCol(h, row, "depot_code"),
			Ligne:        csvCol(h, row, "ligne"),
		})
		return nil
	})
	return out, err
}

func csvCol(header, row []string, name string) string {
	for i, h := range header {
		if h == name && i < len(row) {
			return strings.TrimSpace(row[i])
		}
	}
	return ""
}

func eachCSV(path string, fn func(header, row []string) error) error {
	f, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("scope: %s: %w", filepath.Base(path), err)
	}
	defer f.Close()
	r := csv.NewReader(f)
	r.LazyQuotes = true
	r.FieldsPerRecord = -1
	header, err := r.Read()
	if err != nil {
		return err
	}
	for i := range header {
		header[i] = strings.TrimSpace(header[i])
		if i == 0 {
			header[i] = strings.TrimPrefix(header[i], "\ufeff")
		}
	}
	for {
		row, err := r.Read()
		if err == io.EOF {
			return nil
		}
		if err != nil {
			return err
		}
		if err := fn(header, row); err != nil {
			return err
		}
	}
}
