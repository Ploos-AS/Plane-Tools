const radarCanvas = document.querySelector("#adsb-radar");
const radarRange = document.querySelector("#radar-range");
const radarStatus = document.querySelector("#radar-status");
const radarContext = radarCanvas.getContext("2d");
let radarAircraft = [];

function drawRadar(items) {
  radarAircraft = items;
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

  let plotted = 0;
  for (const item of items) {
    if (item.distance_nm == null || item.bearing_deg == null || item.distance_nm > rangeNM) continue;
    const distanceRatio = item.distance_nm / rangeNM;
    const radians = item.bearing_deg * Math.PI / 180;
    const x = cx + Math.sin(radians) * radius * distanceRatio;
    const y = cy - Math.cos(radians) * radius * distanceRatio;
    const label = item.registration || item.flight || item.hex;

    ctx.save();
    ctx.translate(x, y);
    ctx.rotate(((item.track_deg ?? item.bearing_deg) * Math.PI / 180));
    ctx.fillStyle = "#f0cf78";
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

  radarStatus.textContent = plotted
    ? `${plotted} aircraft plotted inside ${rangeNM} NM.`
    : `No aircraft with receiver-relative position inside ${rangeNM} NM.`;
}

const previousRenderADSBAircraft = renderADSBAircraft;
renderADSBAircraft = function(items) {
  previousRenderADSBAircraft(items);
  drawRadar(items);
};

radarRange.addEventListener("change", () => drawRadar(radarAircraft));
drawRadar([]);
refreshADSB();
