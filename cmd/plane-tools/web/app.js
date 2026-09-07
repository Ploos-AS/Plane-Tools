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

function stat(label, value) {
  return `<div class="stat"><strong>${escapeHTML(value)}</strong><span>${escapeHTML(label)}</span></div>`;
}

bindJSON("#distance-form", "/api/v1/distance", "#distance-result", (body) => `${body.distance_nm} NM · ${body.bearing_deg}° true`);
bindJSON("#convert-form", "/api/v1/convert", "#convert-result", (body) => `${body.value} ${body.to}`);
bindJSON("#aircraft-form", "/api/v1/aircraft", "#aircraft-result", (body) => {
  const name = [body.manufacturer, body.model].filter(Boolean).join(" ");
  return [body.registration, body.icao24, body.type_code, name, body.operator].filter(Boolean).join(" · ");
});

function airportRow(item, distanceNM = null) {
  const ident = item.ident || item.icao || item.iata || "—";
  const detail = [item.icao, item.iata, item.municipality, item.country].filter(Boolean).join(" · ");
  const distance = distanceNM == null ? "" : `<small>${escapeHTML(distanceNM)} NM</small>`;
  return `<div class="list-row airport-row"><div><strong>${escapeHTML(item.name)}</strong><small>${escapeHTML(detail)}</small></div><div><button type="button" class="use-airport" data-ident="${escapeHTML(ident)}">Use ${escapeHTML(ident)}</button>${distance}</div></div>`;
}

function bindAirportUseButtons(container) {
  container.querySelectorAll(".use-airport").forEach((button) => {
    button.addEventListener("click", () => {
      document.querySelector("#location-form [name=airport_ident]").value = button.dataset.ident;
      document.querySelector("#log-form [name=airport_ident]").value = button.dataset.ident;
    });
  });
}

const airportSearchForm = document.querySelector("#airport-search-form");
const airportResults = document.querySelector("#airport-results");
airportSearchForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  airportResults.textContent = "Searching…";
  try {
    const params = new URLSearchParams(new FormData(airportSearchForm));
    params.set("limit", "20");
    const items = await apiJSON(`/api/v1/airports/search?${params}`);
    airportResults.classList.toggle("empty", !items.length);
    airportResults.innerHTML = items.length ? items.map((item) => airportRow(item)).join("") : "No airports found.";
    bindAirportUseButtons(airportResults);
  } catch (error) {
    airportResults.textContent = error.message;
  }
});

const nearbyForm = document.querySelector("#nearby-form");
const nearbyResults = document.querySelector("#nearby-results");
nearbyForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  nearbyResults.textContent = "Searching…";
  try {
    const params = new URLSearchParams(new FormData(nearbyForm));
    const items = await apiJSON(`/api/v1/airports/nearby?${params}`);
    nearbyResults.classList.toggle("empty", !items.length);
    nearbyResults.innerHTML = items.length ? items.map((item) => airportRow(item.airport, item.distance_nm)).join("") : "No airports within radius.";
    bindAirportUseButtons(nearbyResults);
  } catch (error) {
    nearbyResults.textContent = error.message;
  }
});

const locationForm = document.querySelector("#location-form");
const locationResult = document.querySelector("#location-result");
const locationList = document.querySelector("#location-list");
const locationSubmit = document.querySelector("#location-submit");
const locationCancel = document.querySelector("#location-cancel");
const logLocation = document.querySelector("#log-location");
let cachedLocations = [];

function resetLocationForm() {
  locationForm.reset();
  locationForm.elements.id.value = "";
  locationSubmit.textContent = "Add location";
  locationCancel.hidden = true;
}

function renderLocations(items) {
  cachedLocations = items;
  locationList.classList.toggle("empty", !items.length);
  locationList.innerHTML = items.length ? items.map((item) =>
    `<div class="list-row"><div><strong>${escapeHTML(item.name)}</strong><small>${escapeHTML(`${item.latitude_deg}, ${item.longitude_deg}${item.airport_ident ? ` · ${item.airport_ident}` : ""}`)}</small></div><div class="actions"><button type="button" class="edit-location" data-id="${escapeHTML(item.id)}">Edit</button><button type="button" class="delete-location secondary" data-id="${escapeHTML(item.id)}">Delete</button></div></div>`
  ).join("") : "No saved locations.";

  const selected = logLocation.value;
  logLocation.innerHTML = `<option value="">No saved location</option>${items.map((item) => `<option value="${escapeHTML(item.id)}">${escapeHTML(item.name)}${item.airport_ident ? ` · ${escapeHTML(item.airport_ident)}` : ""}</option>`).join("")}`;
  if ([...logLocation.options].some((option) => option.value === selected)) logLocation.value = selected;

  locationList.querySelectorAll(".edit-location").forEach((button) => button.addEventListener("click", () => {
    const item = cachedLocations.find((location) => location.id === button.dataset.id);
    if (!item) return;
    locationForm.elements.id.value = item.id;
    locationForm.elements.name.value = item.name;
    locationForm.elements.latitude_deg.value = item.latitude_deg;
    locationForm.elements.longitude_deg.value = item.longitude_deg;
    locationForm.elements.airport_ident.value = item.airport_ident || "";
    locationSubmit.textContent = "Update location";
    locationCancel.hidden = false;
    locationForm.scrollIntoView({behavior: "smooth", block: "center"});
  }));

  locationList.querySelectorAll(".delete-location").forEach((button) => button.addEventListener("click", async () => {
    try {
      await apiJSON(`/api/v1/spotting-locations/${encodeURIComponent(button.dataset.id)}`, {method: "DELETE"});
      locationResult.textContent = "Location deleted.";
      resetLocationForm();
      await refreshLocations();
    } catch (error) {
      locationResult.textContent = error.message;
    }
  }));
}

async function refreshLocations() {
  try {
    renderLocations(await apiJSON("/api/v1/spotting-locations"));
  } catch (error) {
    locationList.textContent = error.message;
  }
}

locationForm.addEventListener("submit", async (event) => {
  event.preventDefault();
  locationResult.textContent = "Saving…";
  try {
    const raw = Object.fromEntries(new FormData(locationForm));
    const id = raw.id.trim();
    const payload = {
      name: raw.name.trim(),
      latitude_deg: Number(raw.latitude_deg),
      longitude_deg: Number(raw.longitude_deg),
    };
    if (raw.airport_ident.trim()) payload.airport_ident = raw.airport_ident.trim();
    const body = await apiJSON(id ? `/api/v1/spotting-locations/${encodeURIComponent(id)}` : "/api/v1/spotting-locations", {
      method: id ? "PUT" : "POST",
      headers: {"Content-Type": "application/json"},
      body: JSON.stringify(payload),
    });
    locationResult.textContent = `${id ? "Updated" : "Saved"} ${body.name}.`;
    resetLocationForm();
    await refreshLocations();
  } catch (error) {
    locationResult.textContent = error.message;
  }
});

locationCancel.addEventListener("click", resetLocationForm);
document.querySelector("#refresh-locations").addEventListener("click", refreshLocations);

logLocation.addEventListener("change", () => {
  const item = cachedLocations.find((location) => location.id === logLocation.value);
  if (item?.airport_ident) document.querySelector("#log-form [name=airport_ident]").value = item.airport_ident;
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
    await Promise.all([refreshLogbook(), refreshLocations()]);
  } catch (error) {
    logResult.textContent = error.message;
  }
});

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
refreshLocations();
refreshLogbook();
