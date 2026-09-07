const radarCanvas = document.querySelector("#adsb-radar");
const radarRange = document.querySelector("#radar-range");
const radarStatus = document.querySelector("#radar-status");
const radarDetail = document.querySelector("#radar-detail");
const radarTrails = document.querySelector("#radar-trails");
const radarClearTrails = document.querySelector("#radar-clear-trails");
const radarContext = radarCanvas.getContext("2d");
let radarAircraft = [];
let radarHitTargets = [];
let selectedAircraftHex = "";
const radarTrailHistory = new Map();
const radarTrailMaxAgeMS = 2 * 60 * 1000;
const radarTrailMaxPoints = 24;

function aircraftLabel(item) {
  return item.registration || item.flight || item.hex;
}

function prefillSighting(item) {
  logForm.elements.icao24.value = item.hex || "";
  logForm.elements.registration.value = item.registration || "";
  if (item.flight) logForm.elements.notes.value = `ADS-B callsign ${item.flight}`;
  logResult.textContent = `Prefilled from live ADS-B: ${aircraftLabel(item)}`;
  document.querySelector("#add-sighting-card").scrollIntoView({behavior: "smooth", block: "start"});
}

function renderRadarDetail(item) {
  if (!item) {
    radarDetail.className = "radar-detail empty";
    radarDetail.textContent = "Click an aircraft marker or live-list row for details.";
    return;
  }
  const telemetry = [
    item.distance_nm == null ? null : `${item.distance_nm} NM`,
    item.bearing_deg == null ? null : `${item.bearing_deg}° bearing`,
    item.altitude_ft == null ? null : `${item.altitude_ft} ft`,
    item.ground_speed_kt == null ? null : `${item.ground_speed_kt} kt`,
    item.track_deg == null ? null : `${item.track_deg}° track`,
    item.seen_seconds == null ? null : `seen ${item.seen_seconds}s ago`,
  ].filter(Boolean);
  radarDetail.className = "radar-detail";
  radarDetail.innerHTML = `<div><strong>${escapeHTML(aircraftLabel(item))}</strong><small>${escapeHTML([item.flight, item.hex, item.type_code].filter(Boolean).join(" · "))}</small><small>${escapeHTML(telemetry.join(" · "))}</small></div><button id="radar-add-sighting" type="button">Add sighting</button>`;
  document.querySelector("#radar-add-sighting").addEventListener("click", () => prefillSighting(item));
}

function selectAircraft(hex) {
  selectedAircraftHex = hex || "";
  const item = radarAircraft.find((aircraft) => aircraft.hex === selectedAircraftHex) || null;
  renderRadarDetail(item);
  adsbAircraftEl.querySelectorAll(".live-aircraft-row").forEach((row) => {
    const button = row.querySelector(".add-live-sighting");
    row.classList.toggle("selected", Boolean(button && button.dataset.hex === selectedAircraftHex));
  });
  drawRadar(radarAircraft);
}

function updateRadarTrails(items) {
  const now = Date.now();
  for (const [hex, points] of radarTrailHistory) {
    const kept = points.filter((point) => now - point.at <= radarTrailMaxAgeMS).slice(-radarTrailMaxPoints);
    if (kept.length) radarTrailHistory.set(hex, kept);
    else radarTrailHistory.delete(hex);
  }
  for (const item of items) {
    if (!item.hex || item.distance_nm == null || item.bearing_deg == null) continue;
    const points = radarTrailHistory.get(item.hex) || [];
    const last = points[points.length - 1];
    const changed = !last || Math.abs(last.distance_nm - item.distance_nm) >= 0.02 || Math.abs(last.bearing_deg - item.bearing_deg) >= 0.2;
    if (!last || now - last.at >= 4000 || changed) {
      points.push({distance_nm: item.distance_nm, bearing_deg: item.bearing_deg, at: now});
    }
    radarTrailHistory.set(item.hex, points.slice(-radarTrailMaxPoints));
  }
}

function radarPoint(distanceNM, bearingDeg, cx, cy, radius, rangeNM) {
  const ratio = distanceNM / rangeNM;
  const radians = bearingDeg * Math.PI / 180;
  return {
    x: cx + Math.sin(radians) * radius * ratio,
    y: cy - Math.cos(radians) * radius * ratio,
  };
}

function drawTrails(ctx, items, cx, cy, radius, rangeNM) {
  if (!radarTrails.checked) return;
  for (const item of items) {
    const points = radarTrailHistory.get(item.hex) || [];
    const visible = points.filter((point) => point.distance_nm <= rangeNM);
    if (visible.length < 2) continue;
    ctx.save();
    ctx.strokeStyle = item.hex === selectedAircraftHex ? "#ffffff" : "#5c91aa";
    ctx.lineWidth = item.hex === selectedAircraftHex ? 4 : 2;
    ctx.globalAlpha = item.hex === selectedAircraftHex ? 0.9 : 0.55;
    ctx.beginPath();
    visible.forEach((point, index) => {
      const p = radarPoint(point.distance_nm, point.bearing_deg, cx, cy, radius, rangeNM);
      if (index === 0) ctx.moveTo(p.x, p.y);
      else ctx.lineTo(p.x, p.y);
    });
    ctx.stroke();
    ctx.restore();
  }
}

function drawRadar(items) {
  radarAircraft = items;
  radarHitTargets = [];
  const ctx = radarContext;
  const width = radarCanvas.width;
  const height = radarCanvas.height;
  const cx = width / 2;
  const cy = height / 2;
  const radius = Math.min(width, height) * 0.43;
  const rangeNM = Number(radarRange.value);

  ctx.clearRect(0, 0, width, height);
  ctx.fillStyle = "#07131b";
  ctx.fillRect(0, 0, width, height);
  ctx.lineWidth = 2;
  ctx.strokeStyle = "#285266";
  ctx.fillStyle = "#87b9c8";
  ctx.font = "22px ui-sans-serif, system-ui, sans-serif";

  for (let ring = 1; ring <= 4; ring += 1) {
    const r = radius * ring / 4;
    ctx.beginPath();
    ctx.arc(cx, cy, r, 0, Math.PI * 2);
    ctx.stroke();
    ctx.fillText(`${Math.round(rangeNM * ring / 4)} NM`, cx + 8, cy - r + 26);
  }

  for (let bearing = 0; bearing < 360; bearing += 45) {
    const radians = bearing * Math.PI / 180;
    const x = cx + Math.sin(radians) * radius;
    const y = cy - Math.cos(radians) * radius;
    ctx.beginPath();
    ctx.moveTo(cx, cy);
    ctx.lineTo(x, y);
    ctx.stroke();
    const labelX = cx + Math.sin(radians) * (radius + 28);
    const labelY = cy - Math.cos(radians) * (radius + 28);
    ctx.fillText(`${bearing}°`, labelX - 20, labelY + 8);
  }

  ctx.fillStyle = "#d9f1a7";
  ctx.beginPath();
  ctx.arc(cx, cy, 8, 0, Math.PI * 2);
  ctx.fill();
  ctx.fillText("RX", cx + 12, cy - 12);

  drawTrails(ctx, items, cx, cy, radius, rangeNM);

  let plotted = 0;
  for (const item of items) {
    if (item.distance_nm == null || item.bearing_deg == null || item.distance_nm > rangeNM) continue;
    const {x, y} = radarPoint(item.distance_nm, item.bearing_deg, cx, cy, radius, rangeNM);
    const label = aircraftLabel(item);
    const selected = item.hex === selectedAircraftHex;

    radarHitTargets.push({x, y, radius: 20, hex: item.hex});
    if (selected) {
      ctx.strokeStyle = "#ffffff";
      ctx.lineWidth = 4;
      ctx.beginPath();
      ctx.arc(x, y, 18, 0, Math.PI * 2);
      ctx.stroke();
      ctx.strokeStyle = "#285266";
      ctx.lineWidth = 2;
    }

    ctx.save();
    ctx.translate(x, y);
    ctx.rotate(((item.track_deg ?? item.bearing_deg) * Math.PI / 180));
    ctx.fillStyle = selected ? "#ffffff" : "#f0cf78";
    ctx.beginPath();
    ctx.moveTo(0, -10);
    ctx.lineTo(7, 9);
    ctx.lineTo(0, 5);
    ctx.lineTo(-7, 9);
    ctx.closePath();
    ctx.fill();
    ctx.restore();

    ctx.fillStyle = "#eaf0f7";
    ctx.font = "20px ui-sans-serif, system-ui, sans-serif";
    ctx.fillText(label, x + 12, y - 8);
    if (item.altitude_ft != null) {
      ctx.fillStyle = "#8fb7ea";
      ctx.font = "17px ui-sans-serif, system-ui, sans-serif";
      ctx.fillText(`${item.altitude_ft} ft`, x + 12, y + 14);
    }
    plotted += 1;
  }

  const trailText = radarTrails.checked ? " 2-minute trails enabled." : " Trails hidden.";
  radarStatus.textContent = plotted
    ? `${plotted} aircraft plotted inside ${rangeNM} NM.${selectedAircraftHex ? " Selected aircraft highlighted." : ""}${trailText}`
    : `No aircraft with receiver-relative position inside ${rangeNM} NM.${trailText}`;
}

function bindLiveRowSelection() {
  adsbAircraftEl.querySelectorAll(".live-aircraft-row").forEach((row) => {
    const button = row.querySelector(".add-live-sighting");
    if (!button) return;
    row.classList.toggle("selected", button.dataset.hex === selectedAircraftHex);
    row.addEventListener("click", (event) => {
      if (event.target.closest("button")) return;
      selectAircraft(button.dataset.hex);
    });
  });
}

const previousRenderADSBAircraft = renderADSBAircraft;
renderADSBAircraft = function(items) {
  previousRenderADSBAircraft(items);
  updateRadarTrails(items);
  if (selectedAircraftHex && !items.some((item) => item.hex === selectedAircraftHex)) {
    selectedAircraftHex = "";
    renderRadarDetail(null);
  }
  drawRadar(items);
  bindLiveRowSelection();
};

radarCanvas.addEventListener("click", (event) => {
  const rect = radarCanvas.getBoundingClientRect();
  const scaleX = radarCanvas.width / rect.width;
  const scaleY = radarCanvas.height / rect.height;
  const x = (event.clientX - rect.left) * scaleX;
  const y = (event.clientY - rect.top) * scaleY;
  let nearest = null;
  let nearestDistance = Infinity;
  for (const target of radarHitTargets) {
    const distance = Math.hypot(x - target.x, y - target.y);
    if (distance <= target.radius && distance < nearestDistance) {
      nearest = target;
      nearestDistance = distance;
    }
  }
  if (nearest) selectAircraft(nearest.hex);
});

radarCanvas.addEventListener("mousemove", (event) => {
  const rect = radarCanvas.getBoundingClientRect();
  const scaleX = radarCanvas.width / rect.width;
  const scaleY = radarCanvas.height / rect.height;
  const x = (event.clientX - rect.left) * scaleX;
  const y = (event.clientY - rect.top) * scaleY;
  radarCanvas.style.cursor = radarHitTargets.some((target) => Math.hypot(x - target.x, y - target.y) <= target.radius) ? "pointer" : "default";
});

radarRange.addEventListener("change", () => drawRadar(radarAircraft));
radarTrails.addEventListener("change", () => drawRadar(radarAircraft));
radarClearTrails.addEventListener("click", () => {
  radarTrailHistory.clear();
  drawRadar(radarAircraft);
});
drawRadar([]);
refreshADSB();
