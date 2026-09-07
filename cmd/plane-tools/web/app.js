async function bindJSON(formSelector, endpoint, outputSelector, formatter) {
  const form = document.querySelector(formSelector);
  const output = document.querySelector(outputSelector);
  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    output.textContent = "Working…";
    try {
      const params = new URLSearchParams(new FormData(form));
      for (const [key, value] of [...params.entries()]) if (!value) params.delete(key);
      const response = await fetch(`${endpoint}?${params.toString()}`);
      const body = await response.json();
      if (!response.ok) throw new Error(body.error || "Request failed");
      output.textContent = formatter(body);
    } catch (error) {
      output.textContent = error.message;
    }
  });
}

async function apiJSON(url, options = {}) {
  const response = await fetch(url, options);
  const body = response.status === 204 ? null : await response.json();
  if (!response.ok) throw new Error(body?.error || "Request failed");
  return body;
}

function escapeHTML(value) {
  return String(value ?? "").replace(/[&<>"']/g, (char) => ({"&":"&amp;","<":"&lt;",">":"&gt;","\"":"&quot;","'":"&#39;"})[char]);
}

bindJSON("#distance-form", "/api/v1/distance", "#distance-result", (body) => `${body.distance_nm} NM · ${body.bearing_deg}° true`);
bindJSON("#convert-form", "/api/v1/convert", "#convert-result", (body) => `${body.value} ${body.to}`);
bindJSON("#aircraft-form", "/api/v1/aircraft", "#aircraft-result", (body) => {
  const name = [body.manufacturer, body.model].filter(Boolean).join(" ");
  return [body.registration, body.icao24, body.type_code, name, body.operator].filter(Boolean).join(" · ");
});

const logForm = document.querySelector("#log-form");
const logResult = document.querySelector("#log-result");
logForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  logResult.textContent = "Saving…";
  try {
    const data = Object.fromEntries(new FormData(logForm));
    for (const key of Object.keys(data)) data[key] = data[key].trim();
    if (data.observed_at) data.observed_at = new Date(data.observed_at).toISOString();
    else delete data.observed_at;
    for (const key of Object.keys(data)) if (!data[key]) delete data[key];
    const body = await apiJSON("/api/v1/spotting-log", {
      method: "POST",
      headers: {"Content-Type": "application/json"},
      body: JSON.stringify(data),
    });
    logResult.textContent = `Saved ${body.registration || body.icao24} at ${body.observed_at}`;
    logForm.reset();
    await refreshLogbook();
  } catch (error) {
    logResult.textContent = error.message;
  }
});

function stat(label, value) {
  return `<div class="stat"><strong>${escapeHTML(value)}</strong><span>${escapeHTML(label)}</span></div>`;
}

async function refreshLogbook() {
  const summaryEl = document.querySelector("#summary-stats");
  const lifeEl = document.querySelector("#lifelist");
  const recentEl = document.querySelector("#recent-log");
  try {
    const [summary, recent] = await Promise.all([
      apiJSON("/api/v1/spotting-log/summary"),
      apiJSON("/api/v1/spotting-log?limit=20"),
    ]);
    summaryEl.innerHTML = [
      stat("Sightings", summary.total_sightings),
      stat("Unique aircraft", summary.unique_aircraft),
      stat("First", summary.first_observation ? summary.first_observation.slice(0, 10) : "—"),
      stat("Last", summary.last_observation ? summary.last_observation.slice(0, 10) : "—"),
    ].join("");

    lifeEl.classList.toggle("empty", !summary.lifelist.length);
    lifeEl.innerHTML = summary.lifelist.length ? summary.lifelist.map((item) =>
      `<div class="list-row"><div><strong>${escapeHTML(item.registration || item.icao24)}</strong><small>${escapeHTML(item.icao24 || "registration only")}</small></div><div>${item.sightings} sighting${item.sightings === 1 ? "" : "s"}<small>${escapeHTML(item.first_seen.slice(0, 10))}</small></div></div>`
    ).join("") : "No sightings yet.";

    recentEl.classList.toggle("empty", !recent.length);
    recentEl.innerHTML = recent.length ? recent.map((item) =>
      `<div class="list-row"><div><strong>${escapeHTML(item.registration || item.icao24)}</strong><small>${escapeHTML([item.airport_ident, item.spotting_location_id].filter(Boolean).join(" · ") || "No location")}</small></div><div>${escapeHTML(item.observed_at.replace("T", " ").replace("Z", " UTC"))}<small>${escapeHTML(item.notes || "")}</small></div></div>`
    ).join("") : "No sightings yet.";
  } catch (error) {
    summaryEl.innerHTML = stat("Error", error.message);
  }
}

document.querySelector("#refresh-log").addEventListener("click", refreshLogbook);
refreshLogbook();
