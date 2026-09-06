async function bindJSON(formSelector, endpoint, outputSelector, formatter) {
  const form = document.querySelector(formSelector);
  const output = document.querySelector(outputSelector);

  form.addEventListener("submit", async (event) => {
    event.preventDefault();
    output.textContent = "Working…";

    try {
      const params = new URLSearchParams(new FormData(form));
      const response = await fetch(`${endpoint}?${params.toString()}`);
      const body = await response.json();
      if (!response.ok) throw new Error(body.error || "Request failed");
      output.textContent = formatter(body);
    } catch (error) {
      output.textContent = error.message;
    }
  });
}

bindJSON("#distance-form", "/api/v1/distance", "#distance-result", (body) =>
  `${body.distance_nm} NM · ${body.bearing_deg}° true`
);

bindJSON("#convert-form", "/api/v1/convert", "#convert-result", (body) =>
  `${body.value} ${body.to}`
);
