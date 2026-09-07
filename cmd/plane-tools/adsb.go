package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type adsbConfig struct {
	BaseURL string
	Timeout time.Duration
}

var adsbReceiver = adsbConfig{BaseURL: "http://readsb:8080", Timeout: 3 * time.Second}

type adsbAircraft struct {
	Hex      string   `json:"hex"`
	Flight   string   `json:"flight,omitempty"`
	Registration string `json:"registration,omitempty"`
	TypeCode string   `json:"type_code,omitempty"`
	Latitude *float64 `json:"lat,omitempty"`
	Longitude *float64 `json:"lon,omitempty"`
	Altitude *int     `json:"altitude_ft,omitempty"`
	GroundSpeed *float64 `json:"ground_speed_kt,omitempty"`
	Track *float64 `json:"track_deg,omitempty"`
	SeenSeconds *float64 `json:"seen_seconds,omitempty"`
	Messages int64 `json:"messages,omitempty"`
}

type adsbAircraftEnvelope struct {
	Now      float64 `json:"now"`
	Messages int64   `json:"messages"`
	Aircraft []struct {
		Hex      string   `json:"hex"`
		Flight   string   `json:"flight"`
		Registration string `json:"r"`
		TypeCode string   `json:"t"`
		Lat      *float64 `json:"lat"`
		Lon      *float64 `json:"lon"`
		AltBaro  any      `json:"alt_baro"`
		GS       *float64 `json:"gs"`
		Track    *float64 `json:"track"`
		Seen     *float64 `json:"seen"`
		Messages int64    `json:"messages"`
	} `json:"aircraft"`
}

type adsbStatus struct {
	Configured bool   `json:"configured"`
	Reachable  bool   `json:"reachable"`
	BaseURL    string `json:"base_url"`
	Aircraft   int    `json:"aircraft"`
	Messages   int64  `json:"messages,omitempty"`
	GeneratedAt string `json:"generated_at,omitempty"`
	Error      string `json:"error,omitempty"`
}

func configureADSBReceiver(baseURL string, timeout time.Duration) error {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		adsbReceiver = adsbConfig{}
		return nil
	}
	parsed, err := url.Parse(baseURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("PLANE_TOOLS_ADSB_URL must be an absolute http(s) URL")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("PLANE_TOOLS_ADSB_URL must use http or https")
	}
	if timeout <= 0 {
		return fmt.Errorf("ADS-B timeout must be greater than zero")
	}
	adsbReceiver = adsbConfig{BaseURL: strings.TrimRight(baseURL, "/"), Timeout: timeout}
	return nil
}

func adsbAircraftHandler(w http.ResponseWriter, r *http.Request) {
	items, _, err := fetchADSBAircraft(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func adsbStatusHandler(w http.ResponseWriter, r *http.Request) {
	status := adsbStatus{Configured: adsbReceiver.BaseURL != "", BaseURL: adsbReceiver.BaseURL}
	if !status.Configured {
		writeJSON(w, http.StatusOK, status)
		return
	}
	items, envelope, err := fetchADSBAircraft(r.Context())
	if err != nil {
		status.Error = err.Error()
		writeJSON(w, http.StatusOK, status)
		return
	}
	status.Reachable = true
	status.Aircraft = len(items)
	status.Messages = envelope.Messages
	if envelope.Now > 0 {
		status.GeneratedAt = time.Unix(int64(envelope.Now), 0).UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, status)
}

func fetchADSBAircraft(ctx context.Context) ([]adsbAircraft, adsbAircraftEnvelope, error) {
	var envelope adsbAircraftEnvelope
	if adsbReceiver.BaseURL == "" {
		return nil, envelope, fmt.Errorf("ADS-B receiver is not configured")
	}
	ctx, cancel := context.WithTimeout(ctx, adsbReceiver.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adsbReceiver.BaseURL+"/data/aircraft.json", nil)
	if err != nil {
		return nil, envelope, err
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, envelope, fmt.Errorf("fetch ADS-B receiver: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, envelope, fmt.Errorf("ADS-B receiver returned HTTP %d", resp.StatusCode)
	}
	dec := json.NewDecoder(resp.Body)
	if err := dec.Decode(&envelope); err != nil {
		return nil, envelope, fmt.Errorf("decode ADS-B aircraft.json: %w", err)
	}
	items := make([]adsbAircraft, 0, len(envelope.Aircraft))
	for _, raw := range envelope.Aircraft {
		item := adsbAircraft{
			Hex: strings.ToUpper(strings.TrimSpace(raw.Hex)),
			Flight: strings.TrimSpace(raw.Flight),
			Registration: strings.ToUpper(strings.TrimSpace(raw.Registration)),
			TypeCode: strings.ToUpper(strings.TrimSpace(raw.TypeCode)),
			Latitude: raw.Lat,
			Longitude: raw.Lon,
			GroundSpeed: raw.GS,
			Track: raw.Track,
			SeenSeconds: raw.Seen,
			Messages: raw.Messages,
		}
		if altitude, ok := numericAltitude(raw.AltBaro); ok {
			item.Altitude = &altitude
		}
		if item.Hex != "" {
			items = append(items, item)
		}
	}
	return items, envelope, nil
}

func numericAltitude(value any) (int, bool) {
	switch v := value.(type) {
	case float64:
		return int(v), true
	case json.Number:
		i, err := v.Int64()
		return int(i), err == nil
	default:
		return 0, false
	}
}
