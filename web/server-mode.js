export function createServerMode(deps) {
  async function apiFetch(url, init = {}) {
    return fetch(url, {
      credentials: "same-origin",
      ...init,
    });
  }

  async function loadSession() {
    try {
      const response = await apiFetch("/api/session");
      const contentType = response.headers.get("Content-Type") || "";
      if (!response.ok || !contentType.includes("application/json")) {
        deps.enterBrowserMode();
        return true;
      }
      deps.setRuntimeMode("server");
      document.documentElement.dataset.runtime = "server";
      deps.setSessionState(await response.json());
      deps.syncAuthUI();
      return true;
    } catch {
      deps.enterBrowserMode();
      return true;
    }
  }

  function clearSessionUI(message) {
    deps.setSessionState({ auth_required: true, authenticated: false });
    deps.closeRenderSocket();
    deps.historyListNode.innerHTML = `<p class="status">${message}</p>`;
    deps.syncAuthUI();
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
    const token = deps.tokenInput.value.trim();
    if (!token) {
      if (!quiet) {
        deps.statusNode.textContent = "enter token first";
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
        deps.statusNode.textContent = await response.text();
      }
      deps.setSessionState({ auth_required: true, authenticated: false });
      deps.syncAuthUI();
      return false;
    }

    deps.setSessionState(await response.json());
    deps.tokenInput.value = "";
    clearTokenQueryParam();
    deps.syncAuthUI();
    deps.ensureRenderSocket();
    deps.statusNode.textContent = "session unlocked";
    return true;
  }

  async function logoutSession() {
    const response = await apiFetch("/api/session/logout", {
      method: "POST",
    });
    if (!response.ok) {
      deps.statusNode.textContent = await response.text();
      return;
    }

    deps.setSessionState(await response.json());
    deps.closeRenderSocket();
    deps.previewImage.removeAttribute("src");
    deps.syncPreviewState();
    deps.linksNode.innerHTML = "";
    deps.clearCompareState();
    deps.historyListNode.innerHTML = "<p class=\"status\">sign in to view render history</p>";
    deps.statusNode.textContent = "session cleared";
    deps.setRenderProgress(0, "session cleared");
    deps.syncAuthUI();
  }

  async function bootstrapSessionFromURL() {
    const token = new URL(window.location.href).searchParams.get("token");
    if (!token) {
      return false;
    }
    deps.tokenInput.value = token;
    return loginWithToken({ quiet: true });
  }

  async function loadPalettes() {
    if (deps.authLocked()) {
      deps.paletteSelect.innerHTML = "<option>sign in required</option>";
      return;
    }

    const response = await apiFetch("/api/palettes");
    if (response.status === 401) {
      clearSessionUI("sign in to load palettes");
      return;
    }
    if (!response.ok) {
      deps.statusNode.textContent = await response.text();
      return;
    }

    const palettes = await response.json();
    deps.paletteSelect.innerHTML = "";
    for (const palette of palettes) {
      const option = document.createElement("option");
      option.value = palette.key;
      option.textContent = `${palette.display_name} (${palette.colors.join(" ")})`;
      deps.paletteSelect.append(option);
    }
  }

  async function loadHistory() {
    if (deps.authLocked()) {
      deps.historyListNode.innerHTML = "<p class=\"status\">sign in to view render history</p>";
      return;
    }

    const response = await apiFetch("/api/renders?limit=20");
    if (response.status === 401) {
      clearSessionUI("sign in to view render history");
      return;
    }
    if (!response.ok) {
      deps.historyListNode.innerHTML = `<p class="status">${await response.text()}</p>`;
      return;
    }

    const items = await response.json();
    if (items.length === 0) {
      deps.historyListNode.innerHTML = "<p class=\"status\">no renders yet</p>";
      return;
    }

    deps.historyListNode.innerHTML = items.map((item) => `
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
            <span> · </span>
            <a href="${item.compare_url}" target="_blank" rel="noreferrer">compare</a>
            <span class="debug-only"> · </span>
            <a class="debug-only" href="${item.record_url}" target="_blank" rel="noreferrer">record</a>
            ${item.debug_url ? `<span class="debug-only"> · </span><a class="debug-only" href="${item.debug_url}" target="_blank" rel="noreferrer">debug</a>` : ""}
          </p>
        </div>
      </article>
    `).join("");
  }

  async function renderImage() {
    if (deps.authLocked()) {
      deps.statusNode.textContent = "sign in first";
      return;
    }

    const file = deps.fileInput.files?.[0];
    if (!file) {
      deps.statusNode.textContent = "choose an image first";
      return;
    }

    deps.setRenderInFlight(true);
    deps.syncAuthUI();
    deps.ensureRenderSocket();
    deps.setRenderProgress(4, "queueing render", { animate: true });

    const form = new FormData();
    form.set("file", file);
    form.set("client_id", deps.renderSocketClientID);
    form.set("palette_mode", deps.paletteModeSelect.value);
    form.set("palette", deps.paletteSelect.value);
    form.set("mode", deps.modeSelect.value);
    form.set("width", deps.widthInput.value);
    form.set("height", deps.heightInput.value);
    form.set("crop", deps.cropSelect.value);
    form.set("dither", deps.ditherSelect.value);
    form.set("alpha_mode", deps.alphaModeSelect.value);
    form.set("bg_color", deps.bgColorInput.value);
    form.set("brightness", deps.brightnessInput.value);
    form.set("contrast", deps.contrastInput.value);
    form.set("gamma", deps.gammaInput.value);
    form.set("preview_scale", deps.previewScaleInput.value);
    form.set("tile_size", deps.tileSizeInput.value);
    form.set("colors_per_tile", deps.colorsPerTileInput.value);
    form.set("max_palettes", deps.maxPalettesInput.value);
    if (deps.debugInput.checked || deps.modeSelect.value === "cgb-bg") {
      form.set("debug", "1");
    }

    const response = await apiFetch("/api/render", {
      method: "POST",
      body: form,
    });

    if (response.status === 401) {
      clearSessionUI("sign in to render");
      deps.setRenderInFlight(false);
      deps.syncAuthUI();
      deps.setRenderProgress(0, "sign in first");
      return;
    }
    if (!response.ok) {
      deps.setRenderProgress(0, await response.text());
      deps.setRenderInFlight(false);
      deps.syncAuthUI();
      return;
    }

    const payload = await response.json();
    deps.previewImage.src = payload.preview_url;
    deps.syncPreviewState();
    deps.linksNode.innerHTML = `
      <a href="${payload.review_url}" target="_blank" rel="noreferrer">review page</a>
      <span> · </span>
      <a href="${payload.final_url}" target="_blank" rel="noreferrer">final png</a>
      <span> · </span>
      <a href="${payload.compare_url}" target="_blank" rel="noreferrer">compare card</a>
      <span class="debug-only"> · </span>
      <a class="debug-only" href="${payload.record_url}" target="_blank" rel="noreferrer">record json</a>
      ${payload.debug_url ? `<span class="debug-only"> · </span><a class="debug-only" href="${payload.debug_url}" target="_blank" rel="noreferrer">debug sheet</a>` : ""}
    `;
    deps.compareSourceImage.src = payload.source_url;
    deps.comparePixelImage.src = payload.preview_url;
    deps.compareLinksNode.innerHTML = `
      <a href="${payload.source_url}" target="_blank" rel="noreferrer">original</a>
      <span> · </span>
      <a href="${payload.compare_url}" target="_blank" rel="noreferrer">compare card</a>
    `;
    deps.setRenderProgress(100, "render complete");
    deps.setRenderInFlight(false);
    deps.syncAuthUI();
    void loadHistory();
  }

  return {
    bootstrapSessionFromURL,
    loadHistory,
    loadPalettes,
    loadSession,
    loginWithToken,
    logoutSession,
    renderImage,
  };
}

