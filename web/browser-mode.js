import { createObjectURLStore, createWasmClient, decodeFileToImageData } from "/wasm-client.js";

export function createBrowserMode(deps) {
  const urls = createObjectURLStore();
  const history = [];
  const wasm = createWasmClient({
    onProgress: (percent, message) => {
      deps.setRenderProgress(percent, message, { animate: deps.renderInFlight() });
    },
  });

  async function loadPalettes() {
    try {
      const palettes = await wasm.palettes();
      deps.paletteSelect.innerHTML = "";
      for (const palette of palettes) {
        const option = document.createElement("option");
        option.value = palette.key;
        option.textContent = `${palette.display_name} (${palette.colors.join(" ")})`;
        deps.paletteSelect.append(option);
      }
    } catch (error) {
      deps.statusNode.textContent = error.message;
    }
  }

  function loadHistory() {
    if (history.length === 0) {
      deps.historyListNode.innerHTML = "<p class=\"status\">browser-local renders appear here until this tab closes</p>";
      return;
    }

    deps.historyListNode.innerHTML = history.map((item) => `
      <article class="history-item">
        <a href="${item.preview_url}" target="_blank" rel="noreferrer"><img src="${item.preview_url}" alt="Preview for ${item.id}"></a>
        <div>
          <span class="stamp">local render</span>
          <p><strong>${item.mode}</strong> · ${item.width}x${item.height}</p>
          <p>${new Date(item.created_at).toLocaleString()}</p>
          <p class="links">
            <a href="${item.final_url}" download="pixgbc-final.png">final</a>
            <span> · </span>
            <a href="${item.compare_url}" download="pixgbc-compare.png">compare</a>
            ${item.debug_url ? `<span class="debug-only"> · </span><a class="debug-only" href="${item.debug_url}" download="pixgbc-debug.png">debug</a>` : ""}
          </p>
        </div>
      </article>
    `).join("");
  }

  async function renderImage() {
    const file = deps.fileInput.files?.[0];
    if (!file) {
      deps.statusNode.textContent = "choose an image first";
      return;
    }

    deps.setRenderInFlight(true);
    deps.syncAuthUI();
    deps.setRenderProgress(6, "preparing local render", { animate: true });

    try {
      const imageData = await decodeFileToImageData(file);
      const payload = currentRenderPayload(deps, imageData);
      deps.setRenderProgress(16, "running browser render", { animate: true });
      const result = await wasm.render(payload, [imageData.data.buffer]);

      const sourceURL = urls.remember(URL.createObjectURL(file));
      const previewURL = urls.png(result.preview_png);
      const finalURL = urls.png(result.final_png);
      const compareURL = urls.png(result.compare_png);
      const debugURL = result.debug_png ? urls.png(result.debug_png) : "";

      deps.previewImage.src = previewURL;
      deps.syncPreviewState();
      deps.linksNode.innerHTML = `
        <a href="${finalURL}" download="pixgbc-final.png">final png</a>
        <span> · </span>
        <a href="${compareURL}" download="pixgbc-compare.png">compare card</a>
        ${debugURL ? `<span class="debug-only"> · </span><a class="debug-only" href="${debugURL}" download="pixgbc-debug.png">debug sheet</a>` : ""}
      `;
      deps.compareSourceImage.src = sourceURL;
      deps.comparePixelImage.src = previewURL;
      deps.compareLinksNode.innerHTML = `
        <a href="${sourceURL}" target="_blank" rel="noreferrer">original</a>
        <span> · </span>
        <a href="${compareURL}" download="pixgbc-compare.png">compare card</a>
      `;

      history.unshift({
        id: `local-${Date.now()}`,
        created_at: new Date().toISOString(),
        mode: result.mode || deps.modeSelect.value,
        width: result.width || numericValue(deps.widthInput),
        height: result.height || numericValue(deps.heightInput),
        preview_url: previewURL,
        final_url: finalURL,
        compare_url: compareURL,
        debug_url: debugURL,
      });
      history.splice(20);
      loadHistory();
      deps.setRenderProgress(100, "browser render complete");
    } catch (error) {
      deps.setRenderProgress(0, error.message || String(error));
    } finally {
      deps.setRenderInFlight(false);
      deps.syncAuthUI();
    }
  }

  return { loadPalettes, loadHistory, renderImage };
}

function currentRenderPayload(deps, imageData) {
  return {
    rgba: imageData.data,
    source_width: imageData.width,
    source_height: imageData.height,
    palette_mode: deps.paletteModeSelect.value,
    palette: deps.paletteSelect.value,
    mode: deps.modeSelect.value,
    width: numericValue(deps.widthInput),
    height: numericValue(deps.heightInput),
    crop: deps.cropSelect.value,
    dither: deps.ditherSelect.value,
    alpha_mode: deps.alphaModeSelect.value,
    bg_color: deps.bgColorInput.value,
    brightness: numericValue(deps.brightnessInput),
    contrast: numericValue(deps.contrastInput),
    gamma: numericValue(deps.gammaInput),
    preview_scale: numericValue(deps.previewScaleInput),
    tile_size: numericValue(deps.tileSizeInput),
    colors_per_tile: numericValue(deps.colorsPerTileInput),
    max_palettes: numericValue(deps.maxPalettesInput),
    debug: deps.debugInput.checked || deps.modeSelect.value === "cgb-bg",
  };
}

function numericValue(input) {
  return Number(input.value);
}

