package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const adsbMaxResponseBytes int64 = 8 << 20

type adsbConfig struct {
	BaseURL   string
	Timeout   time.Duration
	Latitude  *float64
	Longitude *float64
}

var adsbReceiver = adsbConfig{BaseURL: "http://readsb:8080", Timeout: 3 * time.Second}

type adsbFetchError struct {
	Code string
	Err  error
}

func (e *adsbFetchError) Error() string { return e.Err.Error() }
func (e *adsbFetchError) Unwrap() error { return e.Err }

func newADSBFetcError(code, format string, args ...any) error {
	return &adsbFetchError{Code: code, Err: fmt.Errorf(format, args...)}
}

func adsbErrorCode(err error) string {
	var fetchErr *adsbFetchError
	if errors.As(err, &fetchErr) {
		return fetchErr.Code
	}
	return "receiver_error"
}

type adsbAircraft struct {
	Hex          string   `json:"hex"`
	Flight       string   `json:"flight,omitempty"`
	Registration string   `json:"registration,omitempty"`
	TypeCode     string   `json:"type_code,omitempty"`
	Latitude     *float64 `json:"lat,omitempty"`
	Longitude    *float64 `json:"lon,omitempty"`
	Altitude     *int     `json:"altitude_ft,omitempty"`
	GroundSpeed  *float64 `json:"ground_speed_kt,omitempty"`
	Track        *float64 `json:"track_deg,omitempty"`
	SeenSeconds  *float64 `json:"seen_seconds,omitempty"`
	Messages     int64    `json:"messages,omitempty"`
	DistanceNM   *float64 `json:"distance_nm,omitempty"`
	BearingDeg   *float64 `json:"bearing_deg,omitempty"`
}

type adsbAircraftEnvelope struct {
	Now      float64 `json:"now"`
	Messages int64   `json:"messages"`
	Aircraft []struct {
		Hex          string   `json:"hex"`
		Flight       string   `json:"flight"`
		Registration string   `json:"r"`
		TypeCode     string   `json:"t"`
		Lat          *float64 `json:"lat"`
		Lon          *float64 `json:"lon"`
		AltBaro      any      `json:"alt_baro"`
		Altitude     any      `json:"altitude"`
		GS           *float64 `json:"gs"`
		Speed        *float64 `json:"speed"`
		Track        *float64 `json:"track"`
		Seen         *float64 `json:"seen"`
		Messages     int64    `json:"messages"`
	} `json:"aircraft"`
}

type adsbStatus struct {
	Configured             bool     `json:"configured"`
	Reachable              bool     `json:"reachable"`
	Health                 string   `json:"health"`
	HealthReason           string   `json:"health_reason,omitempty"`
	FeedAgeSeconds         float64  `json:"feed_age_seconds,omitempty"`
	LastSuccessAgeSeconds  float64  `json:"last_success_age_seconds,omitempty"`
	PositionConfigured     bool     `json:"position_configured"`
	ReceiverLatitude       *float64 `json:"receiver_lat,omitempty"`
	ReceiverLongitude      *float64 `json:"receiver_lon,omitempty"`
	Aircraft               int      `json:"aircraft"`
	Messages               int64    `json:"messages,omitempty"`
	GeneratedAt            string   `json:"generated_at,omitempty"`
	ErrorCode              string   `json:"error_code,omitempty"`
	Error                  string   `json:"error,omitempty"`
}

func configureADSBReceiver(baseURL string, timeout time.Duration) error {
	baseURL = strings.TrimSpace(baseURL)
	lat, lon := adsbReceiver.Latitude, adsbReceiver.Longitude
	if baseURL == "" {
		adsbReceiver = adsbConfig{Latitude: lat, Longitude: lon}
		invalidateADSBCache()
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
	adsbReceiver = adsbConfig{BaseURL: strings.TrimRight(baseURL, "/"), Timeout: timeout, Latitude: lat, Longitude: lon}
	invalidateADSBCache()
	return nil
}

func configureADSBPosition(rawLat, rawLon string) error {
	rawLat = strings.TrimSpace(rawLat)
	rawLon = strings.TrimSpace(rawLon)
	if rawLat == "" && rawLon == "" {
		adsbReceiver.Latitude = nil
		adsbReceiver.Longitude = nil
		invalidateADSBCache()
		return nil
	}
	if rawLat == "" || rawLon == "" {
		return fmt.Errorf("PLANE_TOOLS_ADSB_LAT and PLANE_TOOLS_ADSB_LON must be set together")
	}
	lat, err := strconv.ParseFloat(rawLat, 64)
	if err != nil {
		return fmt.Errorf("parse PLANE_TOOLS_ADSB_LAT: %w", err)
	}
	lon, err := strconv.ParseFloat(rawLon, 64)
	if err != nil {
		return fmt.Errorf("parse PLANE_TOOLS_ADSB_LON: %w", err)
	}
	if !validLatLon(lat, lon) {
		return fmt.Errorf("ADS-B receiver coordinates must use latitude -90..90 and longitude -180..180")
	}
	adsbReceiver.Latitude = &lat
	adsbReceiver.Longitude = &lon
	invalidateADSBCache()
	return nil
}

func adsbAircraftHandler(w http.ResponseWriter, r *http.Request) {
	items, _, err := fetchCachedADSBAircraft(r.Context())
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error_code": adsbErrorCode(err), "error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, items)
}

func adsbStatusHandler(w http.ResponseWriter, r *http.Request) {
	positionConfigured := adsbReceiver.Latitude != nil && adsbReceiver.Longitude != nil
	status := adsbStatus{
		Configured: adsbReceiver.BaseURL != "",
		PositionConfigured: positionConfigured, ReceiverLatitude: adsbReceiver.Latitude, ReceiverLongitude: adsbReceiver.Longitude,
	}
	if !status.Configured {
		applyADSBHealth(&status, adsbAircraftEnvelope{}, nil, time.Now())
		writeJSON(w, http.StatusOK, status)
		return
	}
	items, envelope, err := fetchCachedADSBAircraft(r.Context())
	if err != nil {
		status.ErrorCode = adsbErrorCode(err)
		status.Error = err.Error()
		applyADSBHealth(&status, envelope, err, time.Now())
		writeJSON(w, http.StatusOK, status)
		return
	}
	status.Reachable = true
	status.Aircraft = len(items)
	status.Messages = envelope.Messages
	if envelope.Now > 0 {
		status.GeneratedAt = time.Unix(int64(envelope.Now), 0).UTC().Format(time.RFC3339)
	}
	applyADSBHealth(&status, envelope, nil, time.Now())
	writeJSON(w, http.StatusOK, status)
}

func fetchADSBAircraft(ctx context.Context) ([]adsbAircraft, adsbAircraftEnvelope, error) {
	var envelope adsbAircraftEnvelope
	if adsbReceiver.BaseURL == "" {
		return nil, envelope, newADSBFetcError("not_configured", "ADS-B receiver is not configured")
	}
	started := time.Now()
	adsbUpstreamFetches.Add(1)
	defer func() { adsbLastFetchLatencyNS.Store(time.Since(started).Nanoseconds()) }()
	ctx, cancel := context.WithTimeout(ctx, adsbReceiver.Timeout)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, adsbReceiver.BaseURL+"/data/aircraft.json", nil)
	if err != nil {
		adsbUpstreamErrors.Add(1)
		return nil, envelope, newADSBFetcError("request_error", "create ADS-B receiver request: %v", err)
	}
	client := &http.Client{Timeout: adsbReceiver.Timeout}
	resp, err := client.Do(req)
	if err != nil {
		adsbUpstreamErrors.Add(1)
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, envelope, newADSBFetcError("timeout", "ADS-B receiver request timed out")
		}
		return nil, envelope, newADSBFetcError("unreachable", "fetch ADS-B receiver: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		adsbUpstreamErrors.Add(1)
		return nil, envelope, newADSBFetcError("upstream_status", "ADS-B receiver returned HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, adsbMaxResponseBytes+1))
	if err != nil {
		adsbUpstreamErrors.Add(1)
		return nil, envelope, newADSBFetcError("read_error", "read ADS-B aircraft.json: %v", err)
	}
	if int64(len(body)) > adsbMaxResponseBytes {
		adsbUpstreamErrors.Add(1)
		return nil, envelope, newADSBFetcError("response_too_large", "ADS-B aircraft.json exceeds %d bytes", adsbMaxResponseBytes)
	}
	if err := json.Unmarshal(body, &envelope); err != nil {
		adsbUpstreamErrors.Add(1)
		return nil, envelope, newADSBFetcError("invalid_json", "decode ADS-B aircraft.json: %v", err)
	}
	items := make([]adsbAircraft, 0, len(envelope.Aircraft))
	for _, raw := range envelope.Aircraft {
		item := adsbAircraft{
			Hex: strings.ToUpper(strings.TrimSpace(raw.Hex)), Flight: strings.TrimSpace(raw.Flight),
			Registration: strings.ToUpper(strings.TrimSpace(raw.Registration)), TypeCode: strings.ToUpper(strings.TrimSpace(raw.TypeCode)),
			Latitude: raw.Lat, Longitude: raw.Lon, GroundSpeed: raw.GS, Track: raw.Track, SeenSeconds: raw.Seen, Messages: raw.Messages,
		}
		if item.GroundSpeed == nil {
			item.GroundSpeed = raw.Speed
		}
		if altitude, ok := numericAltitude(raw.AltBaro); ok {
			item.Altitude = &altitude
		} else if altitude, ok := numericAltitude(raw.Altitude); ok {
			item.Altitude = &altitude
		}
		if adsbReceiver.Latitude != nil && adsbReceiver.Longitude != nil && item.Latitude != nil && item.Longitude != nil {
			distance := round(greatCircleNM(*adsbReceiver.Latitude, *adsbReceiver.Longitude, *item.Latitude, *item.Longitude), 3)
			bearing := round(initialBearing(*adsbReceiver.Latitude, *adsbReceiver.Longitude, *item.Latitude, *item.Longitude), 2)
			item.DistanceNM = &distance
			item.BearingDeg = &bearing
		}
		if item.Hex != "" {
			items = append(items, item)
		}
	}
	adsbLastSuccessUnixNS.Store(time.Now().UnixNano())
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
