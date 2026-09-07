const adsbDiagnosticsEl = document.querySelector("#adsb-diagnostics");
const adsbDiagnosticsDetailEl = document.querySelector("#adsb-diagnostics-detail");
const adsbDiagnosticsRefreshButton = document.querySelector("#refresh-adsb-diagnostics");
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

  adsbDiagnosticsEl.innerHTML = [
    stat("Health", health),
    stat("Cache hit rate", hitRate),
    stat("Coalesced requests", data.cache_coalesced),
    stat("Upstream errors", `${data.upstream_errors} (${errorRate})`),
    stat("Upstream fetches", data.upstream_fetches),
    stat("Last latency", `${data.last_fetch_latency_ms} ms`),
  ].join("");
  adsbDiagnosticsEl.dataset.health = health;

  const details = [
    `cache ${data.cache_hits} hits / ${data.cache_misses} misses`,
    `TTL ${data.cache_ttl_ms} ms`,
    `last success ${formatDiagnosticTime(data.last_success_at)}`,
  ];
  if (data.feed_age_seconds != null) details.push(`feed age ${data.feed_age_seconds}s`);
  if (data.last_success_age_seconds != null) details.push(`success age ${data.last_success_age_seconds}s`);
  if (data.health_reason) details.push(data.health_reason.replaceAll("_", " "));
  adsbDiagnosticsDetailEl.textContent = details.join(" · ");
}

async function refreshADSBDiagnostics() {
  try {
    renderADSBDiagnostics(await apiJSON("/api/v1/adsb/diagnostics"));
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
