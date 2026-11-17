#!/usr/bin/env node
// Platform-agnostic launcher for getnote binary

'use strict';

const { spawn } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');

const platformNames = { win32: 'windows', darwin: 'darwin', linux: 'linux' };
const archNames = { x64: 'amd64', arm64: 'arm64' };

// Candidate binaries in lookup order:
//   1. bin/<native name>        (installed by postinstall for this OS)
//   2. prebuilt/<os>-<arch>/<native name>  (bundled since v1.5.10)
//   3. bin/getnote.exe          (WSL only, when no native binary exists)
function binaryCandidates(platform = os.platform(), arch = os.arch(), binDir = __dirname, release = os.release()) {
  const binaryName = platform === 'win32' ? 'getnote.exe' : 'getnote';
  const candidates = [path.join(binDir, binaryName)];
  const p = platformNames[platform];
  const a = archNames[arch];
  if (p && a) candidates.push(path.join(binDir, '..', 'prebuilt', `${p}-${a}`, binaryName));
  // Only WSL can use the Windows executable as a last resort.
  if (platform === 'linux' && /microsoft|wsl/i.test(release)) candidates.push(path.join(binDir, 'getnote.exe'));
  return candidates;
}

function resolveBinary(platform, arch, binDir, exists = fs.existsSync, release = os.release()) {
  return binaryCandidates(platform, arch, binDir, release).find((candidate) => exists(candidate)) || null;
}

function main() {
  const platform = os.platform();
  const arch = os.arch();
  const binaryPath = resolveBinary(platform, arch, __dirname);
  if (!binaryPath) {
    console.error(`getnote: no binary for ${platform}/${arch}; looked in:`);
    for (const candidate of binaryCandidates(platform, arch, __dirname)) console.error(`  ${candidate}`);
    console.error('Reinstall with: npm i -g @getnote/cli');
    process.exit(1);
  }

  const fail = (err) => {
    console.error(`getnote: failed to start ${binaryPath}: ${err.message}`);
    process.exit(1);
  };

  // Some failures (e.g. wrong CPU architecture) throw synchronously instead of emitting 'error'.
  let child;
  try {
    child = spawn(binaryPath, process.argv.slice(2), {
      stdio: 'inherit',
      windowsHide: true
    });
  } catch (err) {
    fail(err);
  }

  child.on('error', fail);

  child.on('exit', (code, signal) => {
    if (signal) {
      process.kill(process.pid, signal);
    } else {
      process.exit(code);
    }
  });
}

if (require.main === module) main();

module.exports = { binaryCandidates, resolveBinary };
