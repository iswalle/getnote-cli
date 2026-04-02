#!/usr/bin/env node
// postinstall.js — downloads the getnote binary for the current platform.

'use strict';

const fs = require('fs');
const path = require('path');
const os = require('os');
const { execSync } = require('child_process');

const pkg = require('../package.json');
const VERSION = pkg.version;
const REPO = 'iswalle/getnote-cli';

function getPlatform() {
  const platform = os.platform();
  const arch = os.arch();

  const platformMap = { darwin: 'darwin', linux: 'linux', win32: 'windows' };
  const archMap = { x64: 'amd64', arm64: 'arm64' };

  const p = platformMap[platform];
  const a = archMap[arch];
  if (!p || !a) throw new Error(`Unsupported platform: ${platform}/${arch}`);
  return { platform: p, arch: a };
}

function getBinaryName(platform) {
  return platform.platform === 'windows' ? 'getnote.exe' : 'getnote';
}

function getDownloadURL(platform) {
  const ext = platform.platform === 'windows' ? '.zip' : '.tar.gz';
  const asset = `getnote-cli_${VERSION}_${platform.platform}_${platform.arch}${ext}`;
  return `https://github.com/${REPO}/releases/download/v${VERSION}/${asset}`;
}

async function main() {
  const platform = getPlatform();
  const binDir = path.join(__dirname, '..', 'bin');
  const binaryName = getBinaryName(platform);
  const binaryPath = path.join(binDir, binaryName);
  const url = getDownloadURL(platform);
  const tmpFile = path.join(os.tmpdir(), `getnote-download-${Date.now()}`);

  console.log(`Downloading getnote v${VERSION} for ${platform.platform}/${platform.arch}...`);
  console.log(`URL: ${url}`);

  fs.mkdirSync(binDir, { recursive: true });

  try {
    // Use curl (available on macOS/Linux) or PowerShell (Windows) for reliable redirect handling
    if (platform.platform === 'windows') {
      execSync(
        `powershell -Command "Invoke-WebRequest -Uri '${url}' -OutFile '${tmpFile}'"`,
        { stdio: 'inherit' }
      );
      execSync(`powershell -Command "Expand-Archive -Path '${tmpFile}' -DestinationPath '${binDir}' -Force"`, { stdio: 'inherit' });
    } else {
      execSync(`curl -fsSL "${url}" -o "${tmpFile}"`, { stdio: 'inherit' });
      execSync(`tar -xzf "${tmpFile}" -C "${binDir}" "${binaryName}"`, { stdio: 'inherit' });
    }

    fs.chmodSync(binaryPath, 0o755);
    console.log(`getnote installed at ${binaryPath}`);
  } catch (err) {
    console.error('Failed to install getnote:', err.message);
    process.exit(1);
  } finally {
    try { fs.unlinkSync(tmpFile); } catch (_) {}
  }
}

main();
