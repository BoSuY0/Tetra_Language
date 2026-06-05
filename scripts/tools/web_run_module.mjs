#!/usr/bin/env node

import fs from 'node:fs';
import crypto from 'node:crypto';
import process from 'node:process';

let wasmPath = '';
let surfaceTracePath = '';
for (let i = 2; i < process.argv.length; i++) {
  const arg = process.argv[i];
  if (arg === '--surface-trace') {
    if (i + 1 >= process.argv.length) {
      console.error('usage: node scripts/tools/web_run_module.mjs [--surface-trace trace.json] <module.wasm>');
      process.exit(2);
    }
    surfaceTracePath = process.argv[++i];
    continue;
  }
  if (!wasmPath) {
    wasmPath = arg;
    continue;
  }
  console.error('usage: node scripts/tools/web_run_module.mjs [--surface-trace trace.json] <module.wasm>');
  process.exit(2);
}
if (!wasmPath) {
  console.error('usage: node scripts/tools/web_run_module.mjs [--surface-trace trace.json] <module.wasm>');
  process.exit(2);
}

let instanceRef = null;
const surfaceTrace = {
  schema: 'tetra.surface.web-runner-trace.v1',
  wasm_path: wasmPath,
  frames: [],
};

function memoryView() {
  const memory = instanceRef && instanceRef.exports && instanceRef.exports.memory;
  if (!(memory instanceof WebAssembly.Memory)) {
    throw new Error('tetra_web_v1: missing exported memory');
  }
  return new Uint8Array(memory.buffer);
}

function readUTF8(ptr, len) {
  const view = memoryView();
  const start = ptr >>> 0;
  const end = (ptr + len) >>> 0;
  return new TextDecoder().decode(view.subarray(start, end));
}

function sha256Hex(bytes) {
  return crypto.createHash('sha256').update(bytes).digest('hex');
}

function createSurfaceHost() {
  const surfaces = new Map();
  let nextHandle = 1;
  let clipboard = new Uint8Array([84, 101, 116]);
  return {
    __tetra_surface_open(titlePtr, titleLen, width, height) {
      const title = instanceRef ? readUTF8(titlePtr | 0, titleLen | 0) : '';
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
      const surface = surfaces.get(handle | 0);
      if (!surface || !instanceRef || (eventLen | 0) < 9) {
        return 0;
      }
      const view = new DataView(instanceRef.exports.memory.buffer);
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
      if (!surfaces.has(handle | 0) || !instanceRef || (textLen | 0) < 2) {
        return 0;
      }
      const view = memoryView();
      const start = textPtr >>> 0;
      if (start + 2 > view.length) {
        return 0;
      }
      view[start] = 79;
      view[start + 1] = 75;
      return 2;
    },
    __tetra_surface_clipboard_write_text(handle, textPtr, textLen) {
      if (!surfaces.has(handle | 0) || !instanceRef || (textLen | 0) < 0) {
        return 0;
      }
      const view = memoryView();
      const start = textPtr >>> 0;
      const len = textLen | 0;
      if (start + len > view.length) {
        return 0;
      }
      clipboard = new Uint8Array(view.subarray(start, start + len));
      return len;
    },
    __tetra_surface_clipboard_read_text_into(handle, textPtr, textLen) {
      if (!surfaces.has(handle | 0) || !instanceRef) {
        return 0;
      }
      const view = memoryView();
      const start = textPtr >>> 0;
      const cap = textLen | 0;
      const copied = Math.min(cap, clipboard.length) | 0;
      if (copied < 0 || start + copied > view.length) {
        return 0;
      }
      view.set(clipboard.subarray(0, copied), start);
      return copied;
    },
    __tetra_surface_poll_composition_into(handle, eventPtr, eventLen) {
      if (!surfaces.has(handle | 0) || !instanceRef || (eventLen | 0) < 4) {
        return 0;
      }
      const view = new DataView(instanceRef.exports.memory.buffer);
      const start = eventPtr >>> 0;
      if (start + 16 > view.byteLength) {
        return 0;
      }
      view.setInt32(start, 1, true);
      view.setInt32(start + 4, 1, true);
      view.setInt32(start + 8, 1, true);
      view.setInt32(start + 12, 1, true);
      return 4;
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
      if (surfaceTracePath) {
        const view = memoryView();
        const start = pixelsPtr >>> 0;
        const len = pixelsLen | 0;
        if (len < 0 || start + len > view.length) {
          return 1;
        }
        const pixels = view.slice(start, start + len);
        surfaceTrace.frames.push({
          order: surface.presented,
          width: width | 0,
          height: height | 0,
          stride: stride | 0,
          pixels_len: len,
          checksum: sha256Hex(pixels),
        });
      }
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

try {
  const bytes = fs.readFileSync(wasmPath);
  const result = await WebAssembly.instantiate(bytes, {
    tetra_web_v1: {
      console_log(ptr, len) {
        process.stdout.write(readUTF8(ptr | 0, len | 0));
      },
      panic(code, ptr, len) {
        const message = instanceRef ? readUTF8(ptr | 0, len | 0) : 'panic';
        throw new Error(`tetra panic(${code | 0}): ${message}`);
      },
    },
    tetra_surface_host_v1: createSurfaceHost(),
  });
  instanceRef = result.instance;
  const tetraMain = instanceRef.exports.tetra_main;
  if (typeof tetraMain !== 'function') {
    console.error('web module missing tetra_main export');
    process.exit(1);
  }
  const exitCode = (tetraMain() | 0) & 0xff;
  if (surfaceTracePath) {
    fs.writeFileSync(surfaceTracePath, `${JSON.stringify(surfaceTrace, null, 2)}\n`);
  }
  process.exit(exitCode);
} catch (err) {
  console.error(String(err && err.stack ? err.stack : err));
  process.exit(1);
}
