#!/usr/bin/env node

import fs from 'node:fs';
import process from 'node:process';
import { WASI } from 'node:wasi';

const wasmPath = process.argv[2];
if (!wasmPath) {
  console.error('usage: node scripts/tools/wasi_run_module.mjs <module.wasm>');
  process.exit(2);
}

try {
  const bytes = fs.readFileSync(wasmPath);
  const wasi = new WASI({
    version: 'preview1',
    args: [wasmPath],
    env: process.env,
    preopens: {},
  });
  const module = await WebAssembly.compile(bytes);
  const instance = await WebAssembly.instantiate(module, {
    wasi_snapshot_preview1: wasi.wasiImport,
  });
  if (typeof instance.exports._start !== 'function') {
    console.error('wasi module missing _start export');
    process.exit(1);
  }
  wasi.start(instance);
  process.exit(0);
} catch (err) {
  if (err && typeof err === 'object' && typeof err.code === 'number') {
    const exitCode = Number(err.code) & 0xff;
    process.exit(exitCode);
  }
  console.error(String(err && err.stack ? err.stack : err));
  process.exit(1);
}
