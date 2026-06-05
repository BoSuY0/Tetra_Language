const TETRA_WASM_URL = new URL("surface-tree-app.wasm", import.meta.url);

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

function createSurfaceHost(instanceRef) {
  const surfaces = new Map();
  let nextHandle = 1;
  return {
    __tetra_surface_open(titlePtr, titleLen, width, height) {
      const instance = instanceRef.instance;
      const title = instance ? readUTF8(instance, titlePtr | 0, titleLen | 0) : "";
      const handle = nextHandle++;
      surfaces.set(handle, { title, width: width | 0, height: height | 0, presented: 0 });
      return handle | 0;
    },
    __tetra_surface_close(handle) {
      surfaces.delete(handle | 0);
      return 0;
    },
    __tetra_surface_poll_event_kind(handle) {
      return surfaces.has(handle | 0) ? 5 : 1;
    },
    __tetra_surface_poll_event_x(handle) {
      return surfaces.has(handle | 0) ? 48 : 0;
    },
    __tetra_surface_poll_event_y(handle) {
      return surfaces.has(handle | 0) ? 96 : 0;
    },
    __tetra_surface_poll_event_button(handle) {
      return surfaces.has(handle | 0) ? 1 : 0;
    },
    __tetra_surface_poll_event_into(handle, eventPtr, eventLen) {
      const instance = instanceRef.instance;
      const surface = surfaces.get(handle | 0);
      if (!surface || !instance || (eventLen | 0) < 9) {
        return 0;
      }
      const view = new DataView(instance.exports.memory.buffer);
      const start = eventPtr >>> 0;
      if (start + 36 > view.byteLength) {
        return 0;
      }
      view.setInt32(start, 5, true);
      view.setInt32(start + 4, 48, true);
      view.setInt32(start + 8, 96, true);
      view.setInt32(start + 12, 1, true);
      view.setInt32(start + 16, 0, true);
      view.setInt32(start + 20, surface.width | 0, true);
      view.setInt32(start + 24, surface.height | 0, true);
      view.setInt32(start + 28, 0, true);
      view.setInt32(start + 32, 0, true);
      return 9;
    },
    __tetra_surface_poll_event_text_len(handle) {
      return surfaces.has(handle | 0) ? 2 : 0;
    },
    __tetra_surface_poll_event_text_into(handle, textPtr, textLen) {
      const instance = instanceRef.instance;
      if (!surfaces.has(handle | 0) || !instance || (textLen | 0) < 2) {
        return 0;
      }
      const view = memoryView(instance);
      const start = textPtr >>> 0;
      if (start + 2 > view.length) {
        return 0;
      }
      view[start] = 79;
      view[start + 1] = 75;
      return 2;
    },
    __tetra_surface_begin_frame(handle) {
      return surfaces.has(handle | 0) ? 0 : 1;
    },
    __tetra_surface_present_rgba(handle, pixelsPtr, pixelsLen, width, height, stride) {
      const surface = surfaces.get(handle | 0);
      if (!surface) {
        return 1;
      }
      surface.width = width | 0;
      surface.height = height | 0;
      surface.presented = (surface.presented + 1) | 0;
      surface.lastFrame = { pixelsPtr: pixelsPtr | 0, pixelsLen: pixelsLen | 0, stride: stride | 0 };
      return 0;
    },
    __tetra_surface_now_ms() {
      return 0;
    },
    __tetra_surface_request_redraw(handle) {
      return surfaces.has(handle | 0) ? 0 : 1;
    },
  };
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
    tetra_surface_host_v1: createSurfaceHost(instanceRef),
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
