'use strict';
const assert = require('node:assert/strict');
const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const test = require('node:test');
const { spawnSync } = require('node:child_process');
const { resolveBinary } = require('../bin/getnote.js');

const bin = path.join('/pkg', 'bin');
const only = (...files) => (p) => files.includes(p);

test('native install uses the platform binary', () => {
  const native = path.join(bin, 'getnote');
  assert.equal(resolveBinary('linux', 'x64', bin, only(native, path.join(bin, 'getnote.exe'))), native);
});

test('Windows-installed package used from WSL falls back to getnote.exe', () => {
  const exe = path.join(bin, 'getnote.exe');
  assert.equal(resolveBinary('linux', 'x64', bin, only(exe)), exe);
});

test('falls back to the bundled prebuilt binary', () => {
  const prebuilt = path.join(bin, '..', 'prebuilt', 'linux-arm64', 'getnote');
  assert.equal(resolveBinary('linux', 'arm64', bin, only(prebuilt)), prebuilt);
  const winPrebuilt = path.join(bin, '..', 'prebuilt', 'windows-amd64', 'getnote.exe');
  assert.equal(resolveBinary('win32', 'x64', bin, only(winPrebuilt)), winPrebuilt);
});

test('returns null when nothing exists', () => {
  assert.equal(resolveBinary('linux', 'x64', bin, () => false), null);
  assert.equal(resolveBinary('freebsd', 'x64', bin, () => false), null);
});

test('missing binary exits 1 with a clear message instead of an unhandled ENOENT', () => {
  const dir = fs.mkdtempSync(path.join(os.tmpdir(), 'getnote-launcher-'));
  fs.mkdirSync(path.join(dir, 'bin'));
  fs.copyFileSync(path.join(__dirname, '..', 'bin', 'getnote.js'), path.join(dir, 'bin', 'getnote.js'));
  const r = spawnSync(process.execPath, [path.join(dir, 'bin', 'getnote.js'), '--version'], { encoding: 'utf8' });
  fs.rmSync(dir, { recursive: true, force: true });
  assert.equal(r.status, 1);
  assert.match(r.stderr, /getnote: no binary for /);
  assert.doesNotMatch(r.stderr, /ENOENT/);
});
