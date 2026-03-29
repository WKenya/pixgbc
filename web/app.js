const paletteModeSelect = document.querySelector("#palette-mode");
const tokenInput = document.querySelector("#token");
const loginButton = document.querySelector("#login");
const logoutButton = document.querySelector("#logout");
const authStatusNode = document.querySelector("#auth-status");
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
const consoleScreen = document.querySelector(".console-screen");
const linksNode = document.querySelector("#links");
const guideTitleNode = document.querySelector("#guide-title");
const guideBodyNode = document.querySelector("#guide-body");
const refreshHistoryButton = document.querySelector("#refresh-history");
const historyListNode = document.querySelector("#history-list");
const debugUIStorageKey = "pixgbc.debug-ui";

const guideNotes = [
  {
    title: "Relaxed for nicer image tone",
    body: "Start in relaxed mode when you want the most forgiving output from a photo or detailed illustration.",
  },
  {
    title: "Use cgb-bg for cartridge rules",
    body: "Switch to cgb-bg when you want tile-bank limits, shared palettes, and a stricter handheld look.",
  },
  {
    title: "Raise contrast for muddy photos",
    body: "If the preview looks flat after resize, try a small contrast bump before pushing brightness.",
  },
  {
    title: "Lower gamma for washed scenes",
    body: "When highlights feel chalky, bring gamma down a bit so the palette holds midtone detail.",
  },
  {
    title: "Extract for direct sampling",
    body: "Use palette mode extract when the source already has a strong color identity you want to preserve.",
  },
];

let renderInFlight = false;
let sessionState = {
  auth_required: false,
  authenticated: true,
};
let guideIndex = 0;

function authLocked() {
  return sessionState.auth_required && !sessionState.authenticated;
}

function isLoopbackHost(hostname) {
  return hostname === "localhost" ||
    hostname === "127.0.0.1" ||
    hostname === "::1" ||
    hostname === "[::1]" ||
    hostname === "0.0.0.0" ||
    hostname.endsWith(".localhost");
}

function debugUIEnabled() {
  return isLoopbackHost(window.location.hostname) || window.localStorage.getItem(debugUIStorageKey) === "1";
}

function syncDebugUI() {
  document.documentElement.dataset.debugUi = debugUIEnabled() ? "on" : "off";
}

function toggleDebugUI() {
  if (isLoopbackHost(window.location.hostname)) {
    syncDebugUI();
    return;
  }
  if (debugUIEnabled()) {
    window.localStorage.removeItem(debugUIStorageKey);
    statusNode.textContent = "debug tools hidden";
  } else {
    window.localStorage.setItem(debugUIStorageKey, "1");
    statusNode.textContent = "debug tools visible";
  }
  syncDebugUI();
}

function syncAuthUI() {
  if (!sessionState.auth_required) {
    authStatusNode.textContent = "Open demo. No sign-in required.";
  } else if (sessionState.authenticated) {
    authStatusNode.textContent = "Protected demo. Session active in this browser.";
  } else {
    authStatusNode.textContent = "Protected demo. Enter token to unlock renders and history.";
  }

  const locked = authLocked();
  tokenInput.disabled = !sessionState.auth_required || sessionState.authenticated;
  loginButton.hidden = !sessionState.auth_required || sessionState.authenticated;
  logoutButton.hidden = !sessionState.auth_required || !sessionState.authenticated;
  renderButton.disabled = locked || renderInFlight;
}

function syncPreviewState() {
  consoleScreen.classList.toggle("has-image", Boolean(previewImage.getAttribute("src")));
}

function syncGuideNote() {
  if (!guideTitleNode || !guideBodyNode) {
    return;
  }
  const note = guideNotes[guideIndex % guideNotes.length];
  guideTitleNode.textContent = note.title;
  guideBodyNode.textContent = note.body;
}

function startGuideRotation() {
  syncGuideNote();
  window.setInterval(() => {
    guideIndex = (guideIndex + 1) % guideNotes.length;
    syncGuideNote();
  }, 4800);
}

async function apiFetch(url, init = {}) {
  return fetch(url, {
    credentials: "same-origin",
    ...init,
  });
}

async function loadSession() {
  const response = await apiFetch("/api/session");
  if (!response.ok) {
    authStatusNode.textContent = await response.text();
    return false;
  }
  sessionState = await response.json();
  syncAuthUI();
  return true;
}

function clearSessionUI(message) {
  sessionState = { auth_required: true, authenticated: false };
  historyListNode.innerHTML = `<p class="status">${message}</p>`;
  syncAuthUI();
}

function clearTokenQueryParam() {
  const url = new URL(window.location.href);
  if (!url.searchParams.has("token")) {
    return;
  }
  url.searchParams.delete("token");
  const next = `${url.pathname}${url.search}${url.hash}`;
  window.history.replaceState({}, "", next);
}

async function loginWithToken({ quiet = false } = {}) {
  const token = tokenInput.value.trim();
  if (!token) {
    if (!quiet) {
      statusNode.textContent = "enter token first";
    }
    return false;
  }

  const response = await apiFetch("/api/session/login", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ token }),
  });

  if (!response.ok) {
    if (!quiet) {
      statusNode.textContent = await response.text();
    }
    sessionState = { auth_required: true, authenticated: false };
    syncAuthUI();
    return false;
  }

  sessionState = await response.json();
  tokenInput.value = "";
  clearTokenQueryParam();
  syncAuthUI();
  statusNode.textContent = "session unlocked";
  return true;
}

async function logoutSession() {
  const response = await apiFetch("/api/session/logout", {
    method: "POST",
  });
  if (!response.ok) {
    statusNode.textContent = await response.text();
    return;
  }

  sessionState = await response.json();
  previewImage.removeAttribute("src");
  syncPreviewState();
  linksNode.innerHTML = "";
  historyListNode.innerHTML = "<p class=\"status\">sign in to view render history</p>";
  statusNode.textContent = "session cleared";
  syncAuthUI();
}

async function bootstrapSessionFromURL() {
  const token = new URL(window.location.href).searchParams.get("token");
  if (!token) {
    return false;
  }
  tokenInput.value = token;
  return loginWithToken({ quiet: true });
}

async function loadPalettes() {
  if (authLocked()) {
    paletteSelect.innerHTML = "<option>sign in required</option>";
    return;
  }

  const response = await apiFetch("/api/palettes");
  if (response.status === 401) {
    clearSessionUI("sign in to load palettes");
    return;
  }
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
  if (authLocked()) {
    historyListNode.innerHTML = "<p class=\"status\">sign in to view render history</p>";
    return;
  }

  const response = await apiFetch("/api/renders?limit=20");
  if (response.status === 401) {
    clearSessionUI("sign in to view render history");
    return;
  }
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
        <span class="stamp">saved render</span>
        <p><strong>${item.mode}</strong> · ${item.width}x${item.height}</p>
        <p>${new Date(item.created_at).toLocaleString()}</p>
        <p class="links">
          <a href="${item.review_url}" target="_blank" rel="noreferrer">review</a>
          <span> · </span>
          <a href="${item.final_url}" target="_blank" rel="noreferrer">final</a>
          <span class="debug-only"> · </span>
          <a class="debug-only" href="${item.record_url}" target="_blank" rel="noreferrer">record</a>
          ${item.debug_url ? `<span class="debug-only"> · </span><a class="debug-only" href="${item.debug_url}" target="_blank" rel="noreferrer">debug</a>` : ""}
        </p>
      </div>
    </article>
  `).join("");
}

async function renderImage() {
  if (authLocked()) {
    statusNode.textContent = "sign in first";
    return;
  }

  const file = fileInput.files?.[0];
  if (!file) {
    statusNode.textContent = "choose an image first";
    return;
  }

  renderInFlight = true;
  syncAuthUI();
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

  const response = await apiFetch("/api/render", {
    method: "POST",
    body: form,
  });

  if (response.status === 401) {
    clearSessionUI("sign in to render");
    renderInFlight = false;
    syncAuthUI();
    statusNode.textContent = "sign in first";
    return;
  }
  if (!response.ok) {
    statusNode.textContent = await response.text();
    renderInFlight = false;
    syncAuthUI();
    return;
  }

  const payload = await response.json();
  previewImage.src = payload.preview_url;
  syncPreviewState();
  linksNode.innerHTML = `
    <a href="${payload.review_url}" target="_blank" rel="noreferrer">review page</a>
    <span> · </span>
    <a href="${payload.final_url}" target="_blank" rel="noreferrer">final png</a>
    <span class="debug-only"> · </span>
    <a class="debug-only" href="${payload.record_url}" target="_blank" rel="noreferrer">record json</a>
    ${payload.debug_url ? `<span class="debug-only"> · </span><a class="debug-only" href="${payload.debug_url}" target="_blank" rel="noreferrer">debug sheet</a>` : ""}
  `;
  statusNode.textContent = "render complete";
  renderInFlight = false;
  syncAuthUI();
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

loginButton.addEventListener("click", () => {
  void (async () => {
    if (await loginWithToken()) {
      await loadPalettes();
      await loadHistory();
    }
  })();
});

logoutButton.addEventListener("click", () => {
  void logoutSession();
});

renderButton.addEventListener("click", () => {
  void renderImage();
});

refreshHistoryButton.addEventListener("click", () => {
  void loadHistory();
});

document.addEventListener("keydown", (event) => {
  if (event.repeat || event.metaKey || event.ctrlKey || !event.altKey || !event.shiftKey) {
    return;
  }
  if (event.key.toLowerCase() !== "d") {
    return;
  }
  event.preventDefault();
  toggleDebugUI();
});

paletteModeSelect.addEventListener("change", syncControls);
modeSelect.addEventListener("change", syncControls);

void (async () => {
  syncDebugUI();
  syncPreviewState();
  startGuideRotation();
  const bootstrapped = await bootstrapSessionFromURL();
  if (!bootstrapped) {
    await loadSession();
  }
  await loadPalettes();
  await loadHistory();
  syncControls();
})();
