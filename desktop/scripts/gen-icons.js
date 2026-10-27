#!/usr/bin/env node
// @ts-check
// Generates build/icon.png (512x512) for electron-builder from the same
// rasterizer the app uses at runtime (src/icon.ts), so no binary icon has to
// be committed. Run `npm run build` first (it compiles dist/icon.js).
'use strict';

const fs = require('node:fs');
const path = require('node:path');

const root = path.join(__dirname, '..');
const compiled = path.join(root, 'dist', 'icon.js');
if (!fs.existsSync(compiled)) {
  console.error('dist/icon.js not found - run `npm run build` first.');
  process.exit(1);
}
/** @type {{ appIconPng(size: number): Buffer }} */
const { appIconPng } = require(compiled);

const out = path.join(root, 'build', 'icon.png');
fs.mkdirSync(path.dirname(out), { recursive: true });
const png = appIconPng(512);
fs.writeFileSync(out, png);
console.log(`wrote ${path.relative(root, out)} (${png.length} bytes)`);
