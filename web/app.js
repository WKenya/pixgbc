const paletteSelect = document.querySelector("#palette");
const fileInput = document.querySelector("#file");
const renderButton = document.querySelector("#render");
const statusNode = document.querySelector("#status");
const previewImage = document.querySelector("#preview");

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

  const response = await fetch("/api/render", {
    method: "POST",
    body: form,
  });

  if (!response.ok) {
    statusNode.textContent = await response.text();
    renderButton.disabled = false;
    return;
  }

  const blob = await response.blob();
  previewImage.src = URL.createObjectURL(blob);
  statusNode.textContent = "render complete";
  renderButton.disabled = false;
}

renderButton.addEventListener("click", () => {
  void renderImage();
});

void loadPalettes();
