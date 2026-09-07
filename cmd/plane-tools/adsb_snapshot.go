package main

import (
	"net/http"
	"time"
)

type adsbSnapshot struct {
	Status   adsbStatus     `json:"status"`
	Aircraft []adsbAircraft `json:"aircraft"`
}

func adsbSnapshotHandler(w http.ResponseWriter, r *http.Request) {
	positionConfigured := adsbReceiver.Latitude != nil && adsbReceiver.Longitude != nil
	status := adsbStatus{
		Configured:         adsbReceiver.BaseURL != "",
		PositionConfigured: positionConfigured,
		ReceiverLatitude:   adsbReceiver.Latitude,
		ReceiverLongitude:  adsbReceiver.Longitude,
	}
	if !status.Configured {
		writeJSON(w, http.StatusOK, adsbSnapshot{Status: status, Aircraft: []adsbAircraft{}})
		return
	}

	items, envelope, err := fetchCachedADSBAircraft(r.Context())
	if err != nil {
		status.ErrorCode = adsbErrorCode(err)
		status.Error = err.Error()
		writeJSON(w, http.StatusOK, adsbSnapshot{Status: status, Aircraft: []adsbAircraft{}})
		return
	}
	filtered, err := filterADSBAircraft(items, r)
	if err != nil {
		writeError(w, err)
		return
	}

	status.Reachable = true
	status.Aircraft = len(items)
	status.Messages = envelope.Messages
	if envelope.Now > 0 {
		status.GeneratedAt = time.Unix(int64(envelope.Now), 0).UTC().Format(time.RFC3339)
	}
	writeJSON(w, http.StatusOK, adsbSnapshot{Status: status, Aircraft: filtered})
}
