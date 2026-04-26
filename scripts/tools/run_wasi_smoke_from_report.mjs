#!/usr/bin/env node

import { spawnSync } from 'node:child_process';
import fs from 'node:fs';
import path from 'node:path';

function usage() {
  console.error('Usage: node scripts/tools/run_wasi_smoke_from_report.mjs --build-report <path> --out <path> --runner <wasmtime|node-wasi> --work-dir <path>');
}

function parseArgs(argv) {
  const out = {
    buildReport: '',
    outPath: '',
    runner: '',
    workDir: '',
  };
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    switch (arg) {
      case '--build-report':
        out.buildReport = argv[++i] || '';
        break;
      case '--out':
        out.outPath = argv[++i] || '';
        break;
      case '--runner':
        out.runner = argv[++i] || '';
        break;
      case '--work-dir':
        out.workDir = argv[++i] || '';
        break;
      default:
        throw new Error(`unknown argument: ${arg}`);
    }
  }
  if (!out.buildReport || !out.outPath || !out.runner || !out.workDir) {
    throw new Error('--build-report, --out, --runner, and --work-dir are required');
  }
  if (!['wasmtime', 'node-wasi'].includes(out.runner)) {
    throw new Error(`unsupported runner ${JSON.stringify(out.runner)}`);
  }
  return out;
}

function runModule(runner, wasmPath) {
  if (runner === 'wasmtime') {
    return spawnSync('wasmtime', [wasmPath], { encoding: 'utf8' });
  }
  const script = path.join(process.cwd(), 'scripts', 'tools', 'wasi_run_module.mjs');
  return spawnSync('node', [script, wasmPath], { encoding: 'utf8' });
}

function buildModule(srcPath, outPath) {
  return spawnSync('./tetra', ['build', '--target', 'wasm32-wasi', '-o', outPath, srcPath], { encoding: 'utf8' });
}

function firstLine(text) {
  const trimmed = (text || '').trim();
  if (!trimmed) {
    return '';
  }
  return trimmed.split(/\r?\n/)[0];
}

function toExitCode(result) {
  if (typeof result.status === 'number') {
    return Math.max(0, Math.min(255, result.status));
  }
  return 1;
}

function main() {
  let args;
  try {
    args = parseArgs(process.argv.slice(2));
  } catch (err) {
    usage();
    console.error(String(err.message || err));
    process.exit(2);
  }

  const buildReport = JSON.parse(fs.readFileSync(args.buildReport, 'utf8'));
  const cases = Array.isArray(buildReport.cases) ? buildReport.cases : [];
  fs.mkdirSync(args.workDir, { recursive: true });

  const outCases = [];
  let passed = 0;
  let failed = 0;

  for (const c of cases) {
    const item = {
      name: c.name,
      src_path: c.src_path,
      expected_exit: Number(c.expected_exit || 0),
      out_path: '',
      ran: false,
      pass: false,
    };

    if (!c.pass) {
      item.error = c.error || 'build-only smoke failed';
      failed++;
      outCases.push(item);
      continue;
    }
    if (!c.src_path) {
      item.error = 'missing src_path in build report';
      failed++;
      outCases.push(item);
      continue;
    }

    const wasmPath = path.join(args.workDir, `${item.name}.wasm`);
    const buildResult = buildModule(c.src_path, wasmPath);
    item.out_path = wasmPath;
    if (typeof buildResult.status !== 'number' || buildResult.status !== 0) {
      item.error = firstLine(buildResult.stderr) || firstLine(buildResult.stdout) || 'build failed';
      failed++;
      outCases.push(item);
      continue;
    }

    const result = runModule(args.runner, wasmPath);
    const actualExit = toExitCode(result);
    item.actual_exit = actualExit;
    item.ran = true;
    item.pass = actualExit === item.expected_exit;

    if (item.pass) {
      passed++;
    } else {
      const errLine = firstLine(result.stderr) || firstLine(result.stdout) || `unexpected exit ${actualExit}`;
      item.error = errLine;
      failed++;
    }

    outCases.push(item);
  }

  const report = {
    timestamp: new Date().toISOString(),
    target: buildReport.target || 'wasm32-wasi',
    host: buildReport.host || '',
    version: buildReport.version || '',
    git_head: buildReport.git_head || '',
    islands_debug: Boolean(buildReport.islands_debug),
    total: outCases.length,
    passed,
    failed,
    runner: args.runner,
    cases: outCases,
  };

  fs.mkdirSync(path.dirname(args.outPath), { recursive: true });
  fs.writeFileSync(args.outPath, JSON.stringify(report, null, 2) + '\n');

  if (failed > 0) {
    process.exit(1);
  }
}

main();
