const paletteSelect = document.querySelector("#palette");
const modeSelect = document.querySelector("#mode");
const fileInput = document.querySelector("#file");
const renderButton = document.querySelector("#render");
const statusNode = document.querySelector("#status");
const previewImage = document.querySelector("#preview");
const linksNode = document.querySelector("#links");

async function loadPalettes() {
  const response = await fetch("/api/palettes");
  const palettes = await response.json();

  paletteSelect.innerHTML = "";
  for (const palette of palettes) {
    const option = document.createElement("option");
    option.value = palette.key;
    option.textContent = `${palette.display_name} (${palette.colors.join(" ")})`;
    paletteSelect.append(option);
  }
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
  form.set("palette", paletteSelect.value);
  form.set("mode", modeSelect.value);
  if (modeSelect.value === "cgb-bg") {
    form.set("debug", "1");
  }

  const response = await fetch("/api/render", {
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
    ${payload.debug_url ? `<span> · </span><a href="${payload.debug_url}" target="_blank" rel="noreferrer">heatmap</a>` : ""}
  `;
  statusNode.textContent = "render complete";
  renderButton.disabled = false;
}

renderButton.addEventListener("click", () => {
  void renderImage();
});

void loadPalettes();
