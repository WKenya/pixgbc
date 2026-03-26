const paletteModeSelect = document.querySelector("#palette-mode");
const tokenInput = document.querySelector("#token");
const paletteSelect = document.querySelector("#palette");
const modeSelect = document.querySelector("#mode");
const widthInput = document.querySelector("#width");
const heightInput = document.querySelector("#height");
const cropSelect = document.querySelector("#crop");
const ditherSelect = document.querySelector("#dither");
const alphaModeSelect = document.querySelector("#alpha-mode");
const bgColorInput = document.querySelector("#bg-color");
const brightnessInput = document.querySelector("#brightness");
const contrastInput = document.querySelector("#contrast");
const gammaInput = document.querySelector("#gamma");
const previewScaleInput = document.querySelector("#preview-scale");
const tileSizeInput = document.querySelector("#tile-size");
const colorsPerTileInput = document.querySelector("#colors-per-tile");
const maxPalettesInput = document.querySelector("#max-palettes");
const debugInput = document.querySelector("#debug");
const fileInput = document.querySelector("#file");
const renderButton = document.querySelector("#render");
const statusNode = document.querySelector("#status");
const previewImage = document.querySelector("#preview");
const linksNode = document.querySelector("#links");
const refreshHistoryButton = document.querySelector("#refresh-history");
const historyListNode = document.querySelector("#history-list");
const tokenStorageKey = "pixgbc.token";

function activeToken() {
  return tokenInput.value.trim();
}

function withToken(url) {
  const token = activeToken();
  if (!token) return url;
  const separator = url.includes("?") ? "&" : "?";
  return `${url}${separator}token=${encodeURIComponent(token)}`;
}

function restoreToken() {
  const params = new URLSearchParams(window.location.search);
  const tokenFromURL = params.get("token");
  const tokenFromStorage = window.localStorage.getItem(tokenStorageKey);
  tokenInput.value = tokenFromURL || tokenFromStorage || "";
}

function persistToken() {
  const token = activeToken();
  if (token) {
    window.localStorage.setItem(tokenStorageKey, token);
  } else {
    window.localStorage.removeItem(tokenStorageKey);
  }
}

async function loadPalettes() {
  const response = await fetch(withToken("/api/palettes"));
  if (!response.ok) {
    statusNode.textContent = await response.text();
    return;
  }
  const palettes = await response.json();

  paletteSelect.innerHTML = "";
  for (const palette of palettes) {
    const option = document.createElement("option");
    option.value = palette.key;
    option.textContent = `${palette.display_name} (${palette.colors.join(" ")})`;
    paletteSelect.append(option);
  }
}

async function loadHistory() {
  const response = await fetch(withToken("/api/renders?limit=20"));
  if (!response.ok) {
    historyListNode.innerHTML = `<p class="status">${await response.text()}</p>`;
    return;
  }

  const items = await response.json();
  if (items.length === 0) {
    historyListNode.innerHTML = "<p class=\"status\">no renders yet</p>";
    return;
  }

  historyListNode.innerHTML = items.map((item) => `
    <article class="history-item">
      <a href="${item.review_url}" target="_blank" rel="noreferrer"><img src="${item.preview_url}" alt="Preview for ${item.id}"></a>
      <div>
        <p><strong>${item.mode}</strong> · ${item.width}x${item.height}</p>
        <p>${new Date(item.created_at).toLocaleString()}</p>
        <p class="links">
          <a href="${item.review_url}" target="_blank" rel="noreferrer">review</a>
          <span> · </span>
          <a href="${item.final_url}" target="_blank" rel="noreferrer">final</a>
          <span> · </span>
          <a href="${item.record_url}" target="_blank" rel="noreferrer">record</a>
          ${item.debug_url ? `<span> · </span><a href="${item.debug_url}" target="_blank" rel="noreferrer">debug</a>` : ""}
        </p>
      </div>
    </article>
  `).join("");
}

async function renderImage() {
  const file = fileInput.files?.[0];
  if (!file) {
    statusNode.textContent = "choose an image first";
    return;
  }

  renderButton.disabled = true;
  statusNode.textContent = "rendering...";

  const form = new FormData();
  form.set("file", file);
  form.set("palette_mode", paletteModeSelect.value);
  form.set("palette", paletteSelect.value);
  form.set("mode", modeSelect.value);
  form.set("width", widthInput.value);
  form.set("height", heightInput.value);
  form.set("crop", cropSelect.value);
  form.set("dither", ditherSelect.value);
  form.set("alpha_mode", alphaModeSelect.value);
  form.set("bg_color", bgColorInput.value);
  form.set("brightness", brightnessInput.value);
  form.set("contrast", contrastInput.value);
  form.set("gamma", gammaInput.value);
  form.set("preview_scale", previewScaleInput.value);
  form.set("tile_size", tileSizeInput.value);
  form.set("colors_per_tile", colorsPerTileInput.value);
  form.set("max_palettes", maxPalettesInput.value);
  if (debugInput.checked || modeSelect.value === "cgb-bg") {
    form.set("debug", "1");
  }

  const response = await fetch(withToken("/api/render"), {
    method: "POST",
    body: form,
  });

  if (!response.ok) {
    statusNode.textContent = await response.text();
    renderButton.disabled = false;
    return;
  }

  const payload = await response.json();
  previewImage.src = payload.preview_url;
  linksNode.innerHTML = `
    <a href="${payload.review_url}" target="_blank" rel="noreferrer">review page</a>
    <span> · </span>
    <a href="${payload.final_url}" target="_blank" rel="noreferrer">final png</a>
    <span> · </span>
    <a href="${payload.record_url}" target="_blank" rel="noreferrer">record json</a>
    ${payload.debug_url ? `<span> · </span><a href="${payload.debug_url}" target="_blank" rel="noreferrer">debug sheet</a>` : ""}
  `;
  statusNode.textContent = "render complete";
  renderButton.disabled = false;
  void loadHistory();
}

function syncControls() {
  const extractMode = paletteModeSelect.value === "extract";
  const strictMode = modeSelect.value === "cgb-bg";
  paletteSelect.disabled = extractMode;
  for (const element of document.querySelectorAll(".strict-only input")) {
    element.disabled = !strictMode;
  }
  debugInput.checked = debugInput.checked || strictMode;
}

tokenInput.addEventListener("change", () => {
  persistToken();
  void loadPalettes();
  void loadHistory();
});

renderButton.addEventListener("click", () => {
  void renderImage();
});

refreshHistoryButton.addEventListener("click", () => {
  void loadHistory();
});

paletteModeSelect.addEventListener("change", syncControls);
modeSelect.addEventListener("change", syncControls);

restoreToken();
void loadPalettes();
void loadHistory();
syncControls();
