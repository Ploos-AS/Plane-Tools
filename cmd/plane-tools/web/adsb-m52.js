const adsbFilterForm = document.querySelector("#adsb-filter-form");

function adsbFilterQuery() {
  const params = new URLSearchParams(new FormData(adsbFilterForm));
  for (const [key, value] of [...params.entries()]) if (!String(value).trim()) params.delete(key);
  return params.toString();
}

renderADSBAircraft = function(items) {
  adsbAircraftEl.classList.toggle("empty", !items.length);
  adsbAircraftEl.innerHTML = items.length ? items.map((item) => {
    const title = item.registration || item.flight || item.hex;
    const detail = [item.flight, item.hex, item.type_code].filter(Boolean).join(" · ");
    const geometry = item.distance_nm == null ? "" : `${item.distance_nm} NM · ${item.bearing_deg}°`;
    const telemetry = [
      geometry || null,
      formatLiveValue(item.altitude_ft, " ft"),
      formatLiveValue(item.ground_speed_kt, " kt"),
      formatLiveValue(item.track_deg, "° track"),
      item.seen_seconds == null ? null : `seen ${item.seen_seconds}s ago`,
    ].filter(Boolean).join(" · ");
    return `<div class="list-row live-aircraft-row"><div><strong>${escapeHTML(title)}</strong><small>${escapeHTML(detail)}</small><small>${escapeHTML(telemetry)}</small></div><div><button type="button" class="add-live-sighting" data-hex="${escapeHTML(item.hex)}" data-registration="${escapeHTML(item.registration || "")}" data-flight="${escapeHTML(item.flight || "")}">Add sighting</button></div></div>`;
  }).join("") : "No live aircraft match the current filters.";

  adsbAircraftEl.querySelectorAll(".add-live-sighting").forEach((button) => button.addEventListener("click", () => {
    logForm.elements.icao24.value = button.dataset.hex || "";
    logForm.elements.registration.value = button.dataset.registration || "";
    if (button.dataset.flight) logForm.elements.notes.value = `ADS-B callsign ${button.dataset.flight}`;
    logResult.textContent = `Prefilled from live ADS-B: ${button.dataset.registration || button.dataset.hex}`;
    document.querySelector("#add-sighting-card").scrollIntoView({behavior: "smooth", block: "start"});
  }));
};

function renderReceiverHealth(status) {
  const health = status.health || (status.reachable ? "healthy" : "offline");
  const labels = {healthy: "Healthy", degraded: "Degraded", stale: "Stale", offline: "Offline"};
  const details = [];
  if (status.feed_age_seconds != null) details.push(`feed ${status.feed_age_seconds}s old`);
  if (status.last_success_age_seconds != null) details.push(`last success ${status.last_success_age_seconds}s ago`);
  if (status.stale_data) details.push(`showing cached data ${status.stale_data_age_seconds ?? 0}s old`);
  if (status.health_reason) details.push(status.health_reason.replaceAll("_", " "));
  adsbStatusEl.className = `receiver-status ${health}`;
  return `${labels[health] || health}${details.length ? ` · ${details.join(" · ")}` : ""}`;
}

refreshADSB = async function() {
  try {
    const query = adsbFilterQuery();
    const snapshot = await apiJSON(`/api/v1/adsb/snapshot${query ? `?${query}` : ""}`);
    const status = snapshot.status;
    const healthText = renderReceiverHealth(status);
    if (!status.configured) {
      adsbStatusEl.textContent = `${healthText} · Receiver not configured. Set PLANE_TOOLS_ADSB_URL to enable live aircraft.`;
      adsbAircraftEl.className = "list empty";
      adsbAircraftEl.textContent = "ADS-B receiver is disabled.";
      return;
    }
    if (!status.reachable && !status.stale_data) {
      const reason = status.error_code ? `${status.error_code}: ${status.error || "receiver request failed"}` : (status.error || "receiver request failed");
      adsbStatusEl.textContent = `${healthText} · ${reason}`;
      adsbAircraftEl.className = "list empty";
      adsbAircraftEl.textContent = "Could not load live aircraft.";
      return;
    }
    const geometry = status.position_configured
      ? " · receiver position configured"
      : " · set PLANE_TOOLS_ADSB_LAT/LON for distance";
    const sourceText = status.stale_data ? "cached aircraft" : "aircraft";
    adsbStatusEl.textContent = `${healthText} · ${status.aircraft} ${sourceText}${geometry}`;
    renderADSBAircraft(snapshot.aircraft || []);
  } catch (error) {
    adsbStatusEl.textContent = error.message;
    adsbStatusEl.className = "receiver-status offline";
  }
};

adsbFilterForm.addEventListener("input", refreshADSB);
adsbFilterForm.addEventListener("change", refreshADSB);
scheduleADSBRefresh();
refreshADSB();
