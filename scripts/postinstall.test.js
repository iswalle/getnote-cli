'use strict';

const assert = require('node:assert/strict');
const test = require('node:test');

const { getWindowsExtractArgs, installArchive } = require('./postinstall');

test('Windows extraction invokes Expand-Archive inside a script block', () => {
  assert.deepEqual(getWindowsExtractArgs('C:\\Temp\\getnote.zip', 'C:\\Program Files\\getnote'), [
    '-NoProfile',
    '-Command',
    '& { Expand-Archive -LiteralPath $args[0] -DestinationPath $args[1] -Force }',
    'C:\\Temp\\getnote.zip',
    'C:\\Program Files\\getnote',
  ]);
});

test('temporary archive is removed when installation fails', async () => {
  const removed = [];
  await assert.rejects(
    installArchive({
      platform: { platform: 'windows', arch: 'amd64' },
      binDir: 'C:\\getnote\\bin',
      binaryName: 'getnote.exe',
      binaryPath: 'C:\\getnote\\bin\\getnote.exe',
      url: 'https://example.invalid/getnote.zip',
      tmpFile: 'C:\\Temp\\getnote-download.zip',
      downloadFn: async () => { throw new Error('download failed'); },
      verifyChecksumFn: async () => {},
      runFn: () => {},
      chmodFn: () => {},
      unlinkFn: file => removed.push(file),
    }),
    /download failed/,
  );
  assert.deepEqual(removed, ['C:\\Temp\\getnote-download.zip']);
});
