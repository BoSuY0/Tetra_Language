#!/usr/bin/env node

import crypto from 'node:crypto';
import fs from 'node:fs';
import path from 'node:path';

function usage() {
  console.error('Usage: node scripts/tools/api_diff_report.mjs --docs <path> [--baseline <path>] [--diff-out <path>] [--write-baseline] [--enforce <none|no-breaking|no-change>]');
}

function parseArgs(argv) {
  const out = {
    docs: '',
    baseline: '',
    diffOut: '',
    writeBaseline: false,
    enforce: 'none',
  };
  for (let i = 0; i < argv.length; i++) {
    const arg = argv[i];
    switch (arg) {
      case '--docs':
        out.docs = argv[++i] || '';
        break;
      case '--baseline':
        out.baseline = argv[++i] || '';
        break;
      case '--diff-out':
        out.diffOut = argv[++i] || '';
        break;
      case '--write-baseline':
        out.writeBaseline = true;
        break;
      case '--enforce':
        out.enforce = argv[++i] || '';
        break;
      default:
        throw new Error(`unknown argument: ${arg}`);
    }
  }
  if (!out.docs) {
    throw new Error('--docs is required');
  }
  if (!['none', 'no-breaking', 'no-change'].includes(out.enforce)) {
    throw new Error(`invalid --enforce mode ${JSON.stringify(out.enforce)}`);
  }
  if (!out.baseline && (out.writeBaseline || out.diffOut || out.enforce !== 'none')) {
    throw new Error('--baseline is required when writing or enforcing baseline/diff checks');
  }
  return out;
}

function sha256Hex(textOrBytes) {
  return crypto.createHash('sha256').update(textOrBytes).digest('hex');
}

function normalizeEntry(text) {
  const trimmed = text.trim();
  if (trimmed.startsWith('`') && trimmed.endsWith('`') && trimmed.length >= 2) {
    return trimmed.slice(1, -1);
  }
  return trimmed;
}

function entryIdentity(section, entry) {
  const trimmed = entry.trim();
  const functionMatch = /^(?:async\s+)?func\s+([A-Za-z_][A-Za-z0-9_]*)\s*\(/.exec(trimmed);
  if (functionMatch && (section === 'Functions' || section === 'Tests')) {
    return `func ${functionMatch[1]}`;
  }
  const keywordMatch = /^(const|val|var|struct|enum|protocol|extension|impl|state|view)\s+([^(:\s]+)/.exec(trimmed);
  if (keywordMatch) {
    return `${keywordMatch[1]} ${keywordMatch[2]}`;
  }
  return trimmed;
}

function symbolID(symbol) {
  return `${symbol.module}::${symbol.section}::${entryIdentity(symbol.section, symbol.entry)}`;
}

function parseAPIDocs(md) {
  const lines = md.split(/\r?\n/);
  if (lines.length === 0 || lines[0].trim() !== '# Tetra API Docs') {
    throw new Error('missing # Tetra API Docs heading');
  }

  let metadata = null;
  for (let i = 1; i < lines.length; i++) {
    const line = lines[i].trim();
    if (!line) {
      continue;
    }
    if (line.startsWith('## ')) {
      break;
    }
    if (!line.startsWith('<!-- tetra-api-metadata:') || !line.endsWith('-->')) {
      continue;
    }
    const raw = line
      .slice('<!-- tetra-api-metadata:'.length, line.length - '-->'.length)
      .trim();
    metadata = JSON.parse(raw);
    break;
  }
  if (!metadata) {
    throw new Error('missing tetra-api-metadata');
  }

  let moduleName = '';
  let sectionName = '';
  const symbols = [];
  const surfaceLines = [];

  for (const rawLine of lines) {
    const line = rawLine.trim();
    if (line.startsWith('## ') && !line.startsWith('### ')) {
      moduleName = line.slice(3).trim();
      sectionName = '';
      if (moduleName) {
        surfaceLines.push(line);
      }
      continue;
    }
    if (line.startsWith('### ')) {
      sectionName = line.slice(4).trim();
      continue;
    }
    if (line.startsWith('- ')) {
      if (!moduleName || !sectionName) {
        continue;
      }
      const entry = normalizeEntry(line.slice(2));
      const id = `${moduleName}::${sectionName}::${entryIdentity(sectionName, entry)}`;
      const symbolHash = `sha256:${sha256Hex(`${moduleName}\n${sectionName}\n${entry}`)}`;
      symbols.push({
        id,
        module: moduleName,
        section: sectionName,
        entry,
        symbol_hash: symbolHash,
      });
      surfaceLines.push(`- ${line.slice(2).trim()}`);
    }
  }

  symbols.sort((a, b) => a.id.localeCompare(b.id));

  return {
    metadata,
    symbols,
    sourceDocsSHA256: `sha256:${sha256Hex(Buffer.from(md, 'utf8'))}`,
    apiSurfaceSHA256: `sha256:${sha256Hex(surfaceLines.join('\n'))}`,
  };
}

function buildBaseline(parsed, docsPath) {
  return {
    schema: 'tetra.api.diff-baseline.v1alpha1',
    created_at: new Date().toISOString(),
    source_docs: docsPath,
    source_docs_sha256: parsed.sourceDocsSHA256,
    api_metadata: {
      schema: parsed.metadata.schema,
      api_hash: parsed.metadata.api_hash,
      module_count: parsed.metadata.module_count,
      entry_count: parsed.metadata.entry_count,
    },
    symbols: parsed.symbols,
  };
}

function reviewMetadata(kind) {
  switch (kind) {
    case 'added':
      return {
        review_status: 'addition_requires_scope_review',
        review_note: 'New API surface; confirm it is intentional v1 scope before updating the baseline.',
      };
    case 'removed':
      return {
        review_status: 'breaking_requires_review',
        review_note: 'Previously baselined API surface was removed; restore it or approve a breaking baseline update.',
      };
    case 'changed':
      return {
        review_status: 'breaking_requires_review',
        review_note: 'Baselined API signature or metadata changed; restore it or approve a deliberate compatibility decision.',
      };
    default:
      return {
        review_status: 'unknown_requires_review',
        review_note: 'Unrecognized API diff kind; inspect before release.',
      };
  }
}

function buildReviewSummary(totalChanges) {
  return {
    status: totalChanges === 0 ? 'clean' : 'needs_review',
    release_checklist: 'docs/checklists/v1_0_release_gate.md',
    baseline_policy: 'docs/spec/api_diff_policy.md',
    checklist: [
      'Classify added entries as intentional v1 scope or remove them before baseline update.',
      'Classify changed entries as deliberate compatibility decisions or restore the previous signature.',
      'Classify removed entries as deliberate breaking decisions or restore the missing API surface.',
      'Update docs/baselines/api-diff-baseline.v1alpha1.json only after review approval.',
    ],
  };
}

function buildDiff(baseline, parsed, baselinePath, candidatePath) {
  const baseSymbols = new Map();
  const candSymbols = new Map();

  for (const symbol of baseline.symbols || []) {
    baseSymbols.set(symbolID(symbol), { ...symbol, id: symbolID(symbol) });
  }
  for (const symbol of parsed.symbols) {
    candSymbols.set(symbolID(symbol), { ...symbol, id: symbolID(symbol) });
  }

  const ids = [...new Set([...baseSymbols.keys(), ...candSymbols.keys()])].sort((a, b) => a.localeCompare(b));

  const changes = [];
  let added = 0;
  let removed = 0;
  let changed = 0;

  for (const id of ids) {
    const before = baseSymbols.get(id);
    const after = candSymbols.get(id);

    if (!before && after) {
      added++;
      changes.push({
        kind: 'added',
        id,
        module: after.module,
        section: after.section,
        before_entry: '',
        after_entry: after.entry,
        before_hash: '',
        after_hash: after.symbol_hash,
        severity: 'minor',
        ...reviewMetadata('added'),
      });
      continue;
    }
    if (before && !after) {
      removed++;
      changes.push({
        kind: 'removed',
        id,
        module: before.module,
        section: before.section,
        before_entry: before.entry,
        after_entry: '',
        before_hash: before.symbol_hash,
        after_hash: '',
        severity: 'major',
        ...reviewMetadata('removed'),
      });
      continue;
    }
    if (!before || !after) {
      continue;
    }
    if (before.symbol_hash !== after.symbol_hash || before.entry !== after.entry) {
      changed++;
      changes.push({
        kind: 'changed',
        id,
        module: after.module,
        section: after.section,
        before_entry: before.entry,
        after_entry: after.entry,
        before_hash: before.symbol_hash,
        after_hash: after.symbol_hash,
        severity: 'major',
        ...reviewMetadata('changed'),
      });
    }
  }

  const baselineMeta = baseline.api_metadata || {};
  const totalChanges = added + removed + changed;

  return {
    schema: 'tetra.api.diff.v1alpha1',
    baseline_path: baselinePath,
    candidate_path: candidatePath,
    baseline: {
      api_hash: baselineMeta.api_hash || '',
      module_count: Number(baselineMeta.module_count || 0),
      entry_count: Number(baselineMeta.entry_count || 0),
    },
    candidate: {
      api_hash: parsed.metadata.api_hash || '',
      module_count: Number(parsed.metadata.module_count || 0),
      entry_count: Number(parsed.metadata.entry_count || 0),
    },
    summary: {
      added,
      removed,
      changed,
    },
    review: buildReviewSummary(totalChanges),
    changes,
  };
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

  const docsRaw = fs.readFileSync(args.docs, 'utf8');
  const parsed = parseAPIDocs(docsRaw);

  if (parsed.metadata.schema !== 'tetra.api.v1alpha1') {
    throw new Error(`unsupported API metadata schema ${JSON.stringify(parsed.metadata.schema)}`);
  }
  if (parsed.metadata.api_hash !== parsed.apiSurfaceSHA256) {
    throw new Error(`api hash mismatch: metadata=${parsed.metadata.api_hash} computed=${parsed.apiSurfaceSHA256}`);
  }
  if (Number(parsed.metadata.entry_count) !== parsed.symbols.length) {
    throw new Error(`entry_count mismatch: metadata=${parsed.metadata.entry_count} parsed=${parsed.symbols.length}`);
  }

  if (args.writeBaseline) {
    const baseline = buildBaseline(parsed, args.docs);
    fs.mkdirSync(path.dirname(args.baseline), { recursive: true });
    fs.writeFileSync(args.baseline, JSON.stringify(baseline, null, 2) + '\n');
  }

  if (!args.baseline) {
    return;
  }

  if (!fs.existsSync(args.baseline)) {
    throw new Error(`baseline not found: ${args.baseline}`);
  }

  const baselineRaw = fs.readFileSync(args.baseline, 'utf8');
  const baseline = JSON.parse(baselineRaw);
  if (baseline.schema !== 'tetra.api.diff-baseline.v1alpha1') {
    throw new Error(`unsupported baseline schema ${JSON.stringify(baseline.schema)}`);
  }

  const diff = buildDiff(baseline, parsed, args.baseline, args.docs);
  if (args.diffOut) {
    fs.mkdirSync(path.dirname(args.diffOut), { recursive: true });
    fs.writeFileSync(args.diffOut, JSON.stringify(diff, null, 2) + '\n');
  }

  const majorChanges = diff.changes.filter((change) => change.severity === 'major').length;
  const totalChanges = diff.summary.added + diff.summary.removed + diff.summary.changed;

  if (args.enforce === 'no-breaking' && majorChanges > 0) {
    throw new Error(`API diff enforcement failed (no-breaking): ${majorChanges} major change(s)`);
  }
  if (args.enforce === 'no-change' && totalChanges > 0) {
    throw new Error(`API diff enforcement failed (no-change): ${totalChanges} change(s)`);
  }
}

try {
  main();
} catch (err) {
  console.error(String(err.message || err));
  process.exit(1);
}
