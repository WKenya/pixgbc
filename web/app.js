import { createBrowserMode } from "/browser-mode.js";
import { createServerMode } from "/server-mode.js";

const paletteModeSelect = document.querySelector("#palette-mode");
const tokenInput = document.querySelector("#token");
const loginButton = document.querySelector("#login");
const logoutButton = document.querySelector("#logout");
const authStatusNode = document.querySelector("#auth-status");
const authStrip = document.querySelector(".auth-strip");
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
const statusProgressNode = document.querySelector("#status-progress");
const statusNode = document.querySelector("#status");
const previewImage = document.querySelector("#preview");
const consoleScreen = document.querySelector(".console-screen");
const linksNode = document.querySelector("#links");
const compareSourceImage = document.querySelector("#compare-source");
const comparePixelImage = document.querySelector("#compare-pixel");
const compareLinksNode = document.querySelector("#compare-links");
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
let runtimeMode = "server";
let sessionState = {
  auth_required: false,
  authenticated: true,
};
let guideIndex = 0;
let paletteModeTouched = false;
let renderSocket = null;
let renderSocketRetryTimer = 0;
const renderSocketClientID = window.crypto?.randomUUID?.() ?? `pixgbc-${Date.now()}-${Math.random().toString(16).slice(2)}`;

const browserMode = createBrowserMode({
  paletteModeSelect,
  tokenInput,
  loginButton,
  logoutButton,
  authStatusNode,
  paletteSelect,
  modeSelect,
  widthInput,
  heightInput,
  cropSelect,
  ditherSelect,
  alphaModeSelect,
  bgColorInput,
  brightnessInput,
  contrastInput,
  gammaInput,
  previewScaleInput,
  tileSizeInput,
  colorsPerTileInput,
  maxPalettesInput,
  debugInput,
  fileInput,
  statusNode,
  previewImage,
  linksNode,
  compareSourceImage,
  comparePixelImage,
  compareLinksNode,
  historyListNode,
  renderInFlight: () => renderInFlight,
  setRenderInFlight: (value) => {
    renderInFlight = value;
  },
  setRenderProgress,
  syncAuthUI,
  syncPreviewState,
});

const serverRuntime = createServerMode({
  paletteModeSelect,
  tokenInput,
  paletteSelect,
  modeSelect,
  widthInput,
  heightInput,
  cropSelect,
  ditherSelect,
  alphaModeSelect,
  bgColorInput,
  brightnessInput,
  contrastInput,
  gammaInput,
  previewScaleInput,
  tileSizeInput,
  colorsPerTileInput,
  maxPalettesInput,
  debugInput,
  fileInput,
  statusNode,
  previewImage,
  linksNode,
  compareSourceImage,
  comparePixelImage,
  compareLinksNode,
  historyListNode,
  renderSocketClientID,
  authLocked,
  clearCompareState,
  closeRenderSocket,
  ensureRenderSocket,
  enterBrowserMode,
  setRenderInFlight: (value) => {
    renderInFlight = value;
  },
  setRuntimeMode: (value) => {
    runtimeMode = value;
  },
  setSessionState: (value) => {
    sessionState = value;
  },
  setRenderProgress,
  syncAuthUI,
  syncPreviewState,
});

function serverMode() {
  return runtimeMode === "server";
}

function authLocked() {
  return serverMode() && sessionState.auth_required && !sessionState.authenticated;
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
  if (!serverMode()) {
    authStrip.hidden = true;
    renderButton.disabled = renderInFlight;
    return;
  }

  authStrip.hidden = false;
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

function setRenderProgress(percent, message, { animate = false } = {}) {
  const clamped = Math.max(0, Math.min(100, Number(percent) || 0));
  statusProgressNode.style.width = `${clamped}%`;
  statusProgressNode.classList.toggle("loading", animate && clamped > 0 && clamped < 100);
  if (message) {
    statusNode.textContent = message;
  }
}

function renderSocketURL() {
  const url = new URL("/ws", window.location.href);
  url.protocol = window.location.protocol === "https:" ? "wss:" : "ws:";
  url.searchParams.set("client_id", renderSocketClientID);
  return url.toString();
}

function closeRenderSocket() {
  if (renderSocketRetryTimer) {
    window.clearTimeout(renderSocketRetryTimer);
    renderSocketRetryTimer = 0;
  }
  if (renderSocket) {
    renderSocket.close();
    renderSocket = null;
  }
}

function handleRenderSocketEvent(event) {
  if (!event || typeof event !== "object") {
    return;
  }
  switch (event.type) {
    case "progress":
      setRenderProgress(event.percent, event.message, { animate: renderInFlight });
      break;
    case "done":
      setRenderProgress(100, event.message || "render complete");
      break;
    case "error":
      setRenderProgress(0, event.message || "render failed");
      break;
    default:
      break;
  }
}

function ensureRenderSocket() {
  if (!serverMode()) {
    return;
  }
  if (authLocked() || renderSocket?.readyState === WebSocket.OPEN || renderSocket?.readyState === WebSocket.CONNECTING) {
    return;
  }
  renderSocket = new WebSocket(renderSocketURL());
  renderSocket.addEventListener("message", (messageEvent) => {
    try {
      handleRenderSocketEvent(JSON.parse(messageEvent.data));
    } catch {
      statusNode.textContent = "socket message error";
    }
  });
  renderSocket.addEventListener("close", () => {
    renderSocket = null;
    if (authLocked()) {
      return;
    }
    if (renderSocketRetryTimer) {
      window.clearTimeout(renderSocketRetryTimer);
    }
    renderSocketRetryTimer = window.setTimeout(() => {
      ensureRenderSocket();
    }, 1200);
  });
}

function clearCompareState() {
  compareSourceImage.removeAttribute("src");
  comparePixelImage.removeAttribute("src");
  compareLinksNode.innerHTML = "";
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

function enterBrowserMode(message = "Browser-local mode. Images stay on this device.") {
  runtimeMode = "browser";
  sessionState = { auth_required: false, authenticated: true };
  document.documentElement.dataset.runtime = "browser";
  closeRenderSocket();
  authStatusNode.textContent = message;
  syncAuthUI();
}

async function loadPalettes() {
  if (!serverMode()) {
    await browserMode.loadPalettes();
    return;
  }
  await serverRuntime.loadPalettes();
}

async function loadHistory() {
  if (!serverMode()) {
    browserMode.loadHistory();
    return;
  }
  await serverRuntime.loadHistory();
}

async function renderImage() {
  if (!serverMode()) {
    await browserMode.renderImage();
    return;
  }
  await serverRuntime.renderImage();
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
    if (await serverRuntime.loginWithToken()) {
      await loadPalettes();
      await loadHistory();
    }
  })();
});

logoutButton.addEventListener("click", () => {
  void serverRuntime.logoutSession();
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

paletteModeSelect.addEventListener("change", () => {
  paletteModeTouched = true;
  syncControls();
});
modeSelect.addEventListener("change", () => {
  if (!paletteModeTouched) {
    paletteModeSelect.value = modeSelect.value === "cgb-bg" ? "extract" : "preset";
  }
  syncControls();
});

void (async () => {
  syncDebugUI();
  syncPreviewState();
  clearCompareState();
  setRenderProgress(0, "Load an image, then render.");
  startGuideRotation();
  await serverRuntime.loadSession();
  if (serverMode()) {
    const bootstrapped = await serverRuntime.bootstrapSessionFromURL();
    if (bootstrapped) {
      await serverRuntime.loadSession();
    }
    ensureRenderSocket();
  }
  await loadPalettes();
  await loadHistory();
  syncControls();
})();
