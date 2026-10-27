#!/usr/bin/env node
// @ts-check
// Runs `node --check` on every shipped/compiled JavaScript file.
'use strict';

const { execFileSync } = require('node:child_process');
const fs = require('node:fs');
const path = require('node:path');

const root = path.join(__dirname, '..');

/** @param {string} dir @returns {string[]} */
function jsFiles(dir) {
  if (!fs.existsSync(dir)) return [];
  return fs.readdirSync(dir, { withFileTypes: true }).flatMap((e) => {
    const p = path.join(dir, e.name);
    if (e.isDirectory()) return jsFiles(p);
    return e.name.endsWith('.js') ? [p] : [];
  });
}

const files = ['dist', 'static', 'scripts'].flatMap((d) => jsFiles(path.join(root, d)));
if (!files.some((f) => f.includes(`${path.sep}dist${path.sep}`))) {
  console.error('no compiled files in dist/ - run `npm run build` first.');
  process.exit(1);
}
let failed = 0;
for (const f of files) {
  try {
    execFileSync(process.execPath, ['--check', f], { stdio: 'pipe' });
  } catch (err) {
    failed++;
    console.error(`FAIL ${path.relative(root, f)}\n${/** @type {any} */ (err).stderr}`);
  }
}
console.log(`node --check: ${files.length - failed}/${files.length} files OK`);
process.exit(failed ? 1 : 0);
