'use strict';
const assert = require('node:assert/strict');
const test = require('node:test');
const { getPlatform, getBinaryName, getPrebuiltPath } = require('./postinstall');

test('maps Windows architectures to bundled executable paths', () => {
  assert.deepEqual(getPlatform('win32', 'x64'), { platform: 'windows', arch: 'amd64' });
  assert.deepEqual(getPlatform('win32', 'arm64'), { platform: 'windows', arch: 'arm64' });
  assert.equal(getBinaryName({ platform: 'windows' }), 'getnote.exe');
  assert.match(getPrebuiltPath({ platform: 'windows', arch: 'amd64' }), /prebuilt[\\/]windows-amd64[\\/]getnote\.exe$/);
});

test('uses a package-local binary path', () => {
  assert.doesNotMatch(getPrebuiltPath({ platform: 'windows', arch: 'amd64' }), /^https?:/);
});
