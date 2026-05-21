#!/usr/bin/env node

import fs from 'node:fs';
import process from 'node:process';

const wasmPath = process.argv[2];
if (!wasmPath) {
  console.error('usage: node scripts/tools/web_run_module.mjs <module.wasm>');
  process.exit(2);
}

let instanceRef = null;

function memoryView() {
  const memory = instanceRef && instanceRef.exports && instanceRef.exports.memory;
  if (!(memory instanceof WebAssembly.Memory)) {
    throw new Error('tetra_web_v0.4.0: missing exported memory');
  }
  return new Uint8Array(memory.buffer);
}

function readUTF8(ptr, len) {
  const view = memoryView();
  const start = ptr >>> 0;
  const end = (ptr + len) >>> 0;
  return new TextDecoder().decode(view.subarray(start, end));
}

try {
  const bytes = fs.readFileSync(wasmPath);
  const result = await WebAssembly.instantiate(bytes, {
    "tetra_web_v0.4.0": {
      console_log(ptr, len) {
        process.stdout.write(readUTF8(ptr | 0, len | 0));
      },
      panic(code, ptr, len) {
        const message = instanceRef ? readUTF8(ptr | 0, len | 0) : 'panic';
        throw new Error(`tetra panic(${code | 0}): ${message}`);
      },
    },
  });
  instanceRef = result.instance;
  const tetraMain = instanceRef.exports.tetra_main;
  if (typeof tetraMain !== 'function') {
    console.error('web module missing tetra_main export');
    process.exit(1);
  }
  process.exit((tetraMain() | 0) & 0xff);
} catch (err) {
  console.error(String(err && err.stack ? err.stack : err));
  process.exit(1);
}
