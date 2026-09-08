const adsbDiagnosticsEl = document.querySelector("#adsb-diagnostics");
const adsbDiagnosticsDetailEl = document.querySelector("#adsb-diagnostics-detail");
const adsbDiagnosticsRefreshButton = document.querySelector("#refresh-adsb-diagnostics");
const adsbHistoryEl = document.querySelector("#adsb-history");
let adsbDiagnosticsTimer = null;

function percent(numerator, denominator) {
  if (!denominator) return "0%";
  return `${Math.round((numerator / denominator) * 100)}%`;
}

function formatDiagnosticTime(value) {
  if (!value) return "never";
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString();
}

function renderADSBDiagnostics(data) {
  const cacheReads = data.cache_hits + data.cache_misses;
  const hitRate = percent(data.cache_hits, cacheReads);
  const errorRate = percent(data.upstream_errors, data.upstream_fetches);
  const health = data.health || "offline";
  const stability = data.stabilizing
    ? "unstable / stabilizing"
    : (data.flapping ? "unstable / flapping" : (data.stability || "stable"));

  adsbDiagnosticsEl.innerHTML = [
    stat("Health", health),
    stat("Stability", stability),
    stat("Cache hit rate", hitRate),
    stat("Coalesced requests", data.cache_coalesced),
    stat("Upstream errors", `${data.upstream_errors} (${errorRate})`),
    stat("Upstream fetches", data.upstream_fetches),
    stat("Last latency", `${data.last_fetch_latency_ms} ms`),
  ].join("");
  adsbDiagnosticsEl.dataset.health = health;
  adsbDiagnosticsEl.dataset.stability = data.flapping ? "flapping" : "stable";

  const details = [
    `cache ${data.cache_hits} hits / ${data.cache_misses} misses`,
    `TTL ${data.cache_ttl_ms} ms`,
    `last success ${formatDiagnosticTime(data.last_success_at)}`,
    `flap transitions ${data.flap_transitions || 0} / ${data.flap_window_seconds || 0}s`,
  ];
  if (data.stabilizing) {
    details.push(`stabilizing ${data.flap_recovery_remaining_seconds || 0}s remaining`);
  } else if (data.flap_recovery_quiet_seconds) {
    details.push(`recovery quiet ${data.flap_recovery_quiet_seconds}s`);
  }
  if (data.feed_age_seconds != null) details.push(`feed age ${data.feed_age_seconds}s`);
  if (data.last_success_age_seconds != null) details.push(`success age ${data.last_success_age_seconds}s`);
  if (data.health_reason) details.push(data.health_reason.replaceAll("_", " "));
  adsbDiagnosticsDetailEl.textContent = details.join(" · ");
}

function renderADSBHistory(events) {
  adsbHistoryEl.classList.toggle("empty", !events.length);
  adsbHistoryEl.innerHTML = events.length ? events.slice(0, 20).map((event) => {
    let title = event.type;
    if (event.type === "health_transition") title = `${event.from_health || "start"} → ${event.to_health}`;
    else if (event.type === "recovery") title = "Receiver recovered";
    else if (event.type === "error") title = "Receiver error";
    const detail = [event.code?.replaceAll("_", " "), formatDiagnosticTime(event.at)].filter(Boolean).join(" · ");
    return `<div class="list-row adsb-history-row"><div><strong>${escapeHTML(title)}</strong><small>${escapeHTML(detail)}</small></div></div>`;
  }).join("") : "No receiver events yet.";
}

async function refreshADSBDiagnostics() {
  try {
    const [diagnostics, history] = await Promise.all([
      apiJSON("/api/v1/adsb/diagnostics"),
      apiJSON("/api/v1/adsb/history"),
    ]);
    renderADSBDiagnostics(diagnostics);
    renderADSBHistory(history);
  } catch (error) {
    adsbDiagnosticsEl.dataset.health = "offline";
    adsbDiagnosticsDetailEl.textContent = `Diagnostics unavailable · ${error.message}`;
  }
}

function scheduleADSBDiagnosticsRefresh() {
  if (adsbDiagnosticsTimer) clearInterval(adsbDiagnosticsTimer);
  adsbDiagnosticsTimer = setInterval(refreshADSBDiagnostics, 5000);
}

adsbDiagnosticsRefreshButton.addEventListener("click", refreshADSBDiagnostics);
scheduleADSBDiagnosticsRefresh();
refreshADSBDiagnostics();
