const TETRA_WASM_URL = new URL("dogfood_web_ui.wasm", import.meta.url);

function memoryView(instance) {
  const memory = instance.exports.memory;
  if (!(memory instanceof WebAssembly.Memory)) {
    throw new Error("tetra_web_v1: missing exported memory");
  }
  return new Uint8Array(memory.buffer);
}

function readUTF8(instance, ptr, len) {
  const view = memoryView(instance);
  const start = ptr >>> 0;
  const end = (ptr + len) >>> 0;
  return new TextDecoder().decode(view.subarray(start, end));
}

function createImports(instanceRef) {
  return {
    tetra_web_v1: {
      console_log(ptr, len) {
        const instance = instanceRef.instance;
        if (!instance) {
          throw new Error("tetra_web_v1: instance is not ready");
        }
        console.log(readUTF8(instance, ptr | 0, len | 0));
      },
      panic(code, ptr, len) {
        const instance = instanceRef.instance;
        let message = "panic";
        if (instance) {
          message = readUTF8(instance, ptr | 0, len | 0);
        }
        throw new Error("tetra panic(" + (code | 0) + "): " + message);
      },
    },
  };
}

export async function instantiateTetra(moduleURL = TETRA_WASM_URL) {
  const response = await fetch(moduleURL);
  if (!response.ok) {
    throw new Error("tetra_web_v1: fetch failed: " + response.status);
  }
  const bytes = await response.arrayBuffer();
  const instanceRef = { instance: null };
  const result = await WebAssembly.instantiate(bytes, createImports(instanceRef));
  instanceRef.instance = result.instance;
  return result;
}

export async function runTetra(moduleURL = TETRA_WASM_URL) {
  const { instance } = await instantiateTetra(moduleURL);
  const tetraMain = instance.exports.tetra_main;
  if (typeof tetraMain !== "function") {
    throw new Error("tetra_web_v1: missing tetra_main export");
  }
  return tetraMain() | 0;
}
