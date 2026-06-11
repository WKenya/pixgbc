importScripts("/wasm_exec.js");

let wasmReady;

function loadWasm() {
  if (wasmReady) {
    return wasmReady;
  }

  wasmReady = (async () => {
    const go = new Go();
    const wasmURL = new URL("/pixgbc.wasm", self.location.href);
    let result;
    try {
      result = await WebAssembly.instantiateStreaming(fetch(wasmURL), go.importObject);
    } catch {
      const response = await fetch(wasmURL);
      if (!response.ok) {
        throw new Error(`load pixgbc.wasm: ${response.status}`);
      }
      const bytes = await response.arrayBuffer();
      result = await WebAssembly.instantiate(bytes, go.importObject);
    }

    void go.run(result.instance);
    await new Promise((resolve) => setTimeout(resolve, 0));
    if (typeof self.pixgbcRender !== "function" || typeof self.pixgbcPalettes !== "function") {
      throw new Error("pixgbc wasm exports missing");
    }
  })();

  return wasmReady;
}

function transferList(payload) {
  const transfers = [];
  for (const key of ["final_png", "preview_png", "compare_png", "debug_png"]) {
    if (payload[key]?.buffer) {
      transfers.push(payload[key].buffer);
    }
  }
  return transfers;
}

self.addEventListener("message", (event) => {
  void (async () => {
    const { id, type, payload } = event.data || {};
    try {
      await loadWasm();
      if (type === "palettes") {
        self.postMessage({ id, type: "palettes", payload: self.pixgbcPalettes() });
        return;
      }
      if (type !== "render") {
        throw new Error(`unknown worker request ${type}`);
      }

      self.postMessage({ id, type: "progress", percent: 18, message: "decoding pixels" });
      const result = self.pixgbcRender(payload);
      if (result?.error) {
        throw new Error(result.error);
      }
      self.postMessage({ id, type: "done", payload: result }, transferList(result));
    } catch (error) {
      self.postMessage({ id, type: "error", message: error?.message || String(error) });
    }
  })();
});

