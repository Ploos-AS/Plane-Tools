package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

//go:embed web/*
var webFS embed.FS

type healthResponse struct {
	Status string `json:"status"`
	Time   string `json:"time"`
}

type distanceResponse struct {
	DistanceNM float64 `json:"distance_nm"`
	BearingDeg float64 `json:"bearing_deg"`
}

type conversionResponse struct {
	Value float64 `json:"value"`
	From  string  `json:"from"`
	To    string  `json:"to"`
}

func main() {
	addr := getenv("PLANE_TOOLS_ADDR", ":8080")

	staticFS, err := fs.Sub(webFS, "web")
	if err != nil {
		log.Fatal(err)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, healthResponse{
			Status: "ok",
			Time:   time.Now().UTC().Format(time.RFC3339),
		})
	})
	mux.HandleFunc("GET /api/v1/distance", distanceHandler)
	mux.HandleFunc("GET /api/v1/convert", convertHandler)
	mux.Handle("/", http.FileServer(http.FS(staticFS)))

	log.Printf("Plane Tools listening on %s", addr)
	if err := http.ListenAndServe(addr, mux); err != nil {
		log.Fatal(err)
	}
}

func distanceHandler(w http.ResponseWriter, r *http.Request) {
	lat1, err := queryFloat(r, "lat1")
	if err != nil {
		writeError(w, err)
		return
	}
	lon1, err := queryFloat(r, "lon1")
	if err != nil {
		writeError(w, err)
		return
	}
	lat2, err := queryFloat(r, "lat2")
	if err != nil {
		writeError(w, err)
		return
	}
	lon2, err := queryFloat(r, "lon2")
	if err != nil {
		writeError(w, err)
		return
	}
	if !validLatLon(lat1, lon1) || !validLatLon(lat2, lon2) {
		writeError(w, fmt.Errorf("coordinates must use latitude -90..90 and longitude -180..180"))
		return
	}

	writeJSON(w, http.StatusOK, distanceResponse{
		DistanceNM: round(greatCircleNM(lat1, lon1, lat2, lon2), 3),
		BearingDeg: round(initialBearing(lat1, lon1, lat2, lon2), 2),
	})
}

func convertHandler(w http.ResponseWriter, r *http.Request) {
	value, err := queryFloat(r, "value")
	if err != nil {
		writeError(w, err)
		return
	}
	from := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("from")))
	to := strings.ToLower(strings.TrimSpace(r.URL.Query().Get("to")))
	result, err := convert(value, from, to)
	if err != nil {
		writeError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, conversionResponse{Value: round(result, 6), From: from, To: to})
}

func greatCircleNM(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusNM = 3440.065
	phi1, phi2 := radians(lat1), radians(lat2)
	dPhi := radians(lat2 - lat1)
	dLambda := radians(lon2 - lon1)

	a := math.Sin(dPhi/2)*math.Sin(dPhi/2) + math.Cos(phi1)*math.Cos(phi2)*math.Sin(dLambda/2)*math.Sin(dLambda/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusNM * c
}

func initialBearing(lat1, lon1, lat2, lon2 float64) float64 {
	phi1, phi2 := radians(lat1), radians(lat2)
	dLambda := radians(lon2 - lon1)
	y := math.Sin(dLambda) * math.Cos(phi2)
	x := math.Cos(phi1)*math.Sin(phi2) - math.Sin(phi1)*math.Cos(phi2)*math.Cos(dLambda)
	degrees := math.Atan2(y, x) * 180 / math.Pi
	return math.Mod(degrees+360, 360)
}

func convert(value float64, from, to string) (float64, error) {
	if from == to && from != "" {
		return value, nil
	}
	const nmToKM = 1.852
	const knotToKPH = 1.852
	const footToM = 0.3048
	switch from + ":" + to {
	case "nm:km":
		return value * nmToKM, nil
	case "km:nm":
		return value / nmToKM, nil
	case "kt:kph":
		return value * knotToKPH, nil
	case "kph:kt":
		return value / knotToKPH, nil
	case "ft:m":
		return value * footToM, nil
	case "m:ft":
		return value / footToM, nil
	default:
		return 0, fmt.Errorf("unsupported conversion %q to %q; supported: nm/km, kt/kph, ft/m", from, to)
	}
}

func queryFloat(r *http.Request, name string) (float64, error) {
	raw := strings.TrimSpace(r.URL.Query().Get(name))
	if raw == "" {
		return 0, fmt.Errorf("missing query parameter %q", name)
	}
	value, err := strconv.ParseFloat(raw, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid %s: %w", name, err)
	}
	return value, nil
}

func validLatLon(lat, lon float64) bool {
	return lat >= -90 && lat <= 90 && lon >= -180 && lon <= 180
}

func radians(v float64) float64 { return v * math.Pi / 180 }

func round(v float64, places int) float64 {
	scale := math.Pow10(places)
	return math.Round(v*scale) / scale
}

func writeError(w http.ResponseWriter, err error) {
	writeJSON(w, http.StatusBadRequest, map[string]string{"error": err.Error()})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func getenv(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}
