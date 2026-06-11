export function createWasmClient({ onProgress } = {}) {
  let worker = null;
  let requestSeq = 0;
  const pending = new Map();

  function ensureWorker() {
    if (worker) {
      return worker;
    }

    worker = new Worker("/wasm-worker.js");
    worker.addEventListener("message", (event) => {
      const message = event.data || {};
      if (message.type === "progress") {
        onProgress?.(message.percent, message.message);
        return;
      }

      const request = pending.get(message.id);
      if (!request) {
        return;
      }
      if (message.type === "error") {
        pending.delete(message.id);
        request.reject(new Error(message.message || "wasm worker error"));
        return;
      }
      if (message.type === "done" || message.type === "palettes") {
        pending.delete(message.id);
        request.resolve(message.payload);
      }
    });
    worker.addEventListener("error", (event) => {
      const error = new Error(event.message || "wasm worker failed");
      for (const request of pending.values()) {
        request.reject(error);
      }
      pending.clear();
    });

    return worker;
  }

  function request(type, payload = {}, transfers = []) {
    const activeWorker = ensureWorker();
    const id = `wasm-${++requestSeq}`;
    return new Promise((resolve, reject) => {
      pending.set(id, { resolve, reject });
      activeWorker.postMessage({ id, type, payload }, transfers);
    });
  }

  return {
    palettes: () => request("palettes"),
    render: (payload, transfers) => request("render", payload, transfers),
  };
}

export async function decodeFileToImageData(file) {
  if (file.type && !["image/png", "image/jpeg"].includes(file.type)) {
    throw new Error("choose a PNG or JPEG image");
  }

  let source;
  let temporaryURL = "";
  if ("createImageBitmap" in window) {
    source = await createImageBitmap(file);
  } else {
    temporaryURL = URL.createObjectURL(file);
    source = await new Promise((resolve, reject) => {
      const image = new Image();
      image.onload = () => resolve(image);
      image.onerror = () => reject(new Error("decode image failed"));
      image.src = temporaryURL;
    });
  }

  try {
    const maxEdge = 4096;
    const maxPixels = 16 * 1024 * 1024;
    const sourceWidth = source.width;
    const sourceHeight = source.height;
    const scale = Math.min(1, maxEdge / sourceWidth, maxEdge / sourceHeight, Math.sqrt(maxPixels / (sourceWidth * sourceHeight)));
    const width = Math.max(1, Math.floor(sourceWidth * scale));
    const height = Math.max(1, Math.floor(sourceHeight * scale));
    const canvas = typeof OffscreenCanvas === "function" ? new OffscreenCanvas(width, height) : document.createElement("canvas");
    canvas.width = width;
    canvas.height = height;
    const context = canvas.getContext("2d", { willReadFrequently: true });
    if (!context) {
      throw new Error("browser canvas unavailable");
    }
    context.clearRect(0, 0, width, height);
    context.drawImage(source, 0, 0, width, height);
    return context.getImageData(0, 0, width, height);
  } finally {
    source.close?.();
    if (temporaryURL) {
      URL.revokeObjectURL(temporaryURL);
    }
  }
}

export function createObjectURLStore(limit = 64) {
  const urls = [];
  return {
    remember(url) {
      urls.push(url);
      while (urls.length > limit) {
        URL.revokeObjectURL(urls.shift());
      }
      return url;
    },
    png(bytes) {
      return this.remember(URL.createObjectURL(new Blob([bytes], { type: "image/png" })));
    },
  };
}

