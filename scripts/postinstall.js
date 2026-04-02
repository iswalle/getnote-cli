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

  // Skip download if binary already matches current version
  if (fs.existsSync(binaryPath)) {
    try {
      const out = execSync(`"${binaryPath}" version 2>/dev/null || "${binaryPath}" --version 2>/dev/null`, { encoding: 'utf8' }).trim();
      if (out.includes(VERSION)) {
        console.log(`getnote v${VERSION} already installed, skipping download.`);
        return;
      }
    } catch (_) { /* binary exists but can't run (wrong arch etc.), re-download */ }
  }

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

    // Create symlink in npm global bin directory so `getnote` is on PATH
    createSymlink(binaryPath, binaryName);
  } catch (err) {
    console.error('Failed to install getnote:', err.message);
    process.exit(1);
  } finally {
    try { fs.unlinkSync(tmpFile); } catch (_) {}
  }
}

main();

// Create a symlink in the npm global bin directory after download.
// npm only creates the symlink at install time when the file already exists;
// since we download the binary in postinstall, we need to do it ourselves.
function createSymlink(binaryPath, binaryName) {
  try {
    const npmPrefix = execSync('npm prefix -g', { encoding: 'utf8' }).trim();
    const globalBin = path.join(npmPrefix, 'bin');
    const symlinkPath = path.join(globalBin, binaryName);

    if (fs.existsSync(globalBin)) {
      try { fs.unlinkSync(symlinkPath); } catch (_) {}
      fs.symlinkSync(binaryPath, symlinkPath);
      console.log(`Symlink created: ${symlinkPath} -> ${binaryPath}`);
    }
  } catch (err) {
    // Non-fatal: user can run the binary directly or add it to PATH manually
    console.warn(`Could not create symlink (you may need to add ${path.dirname(binaryPath)} to PATH): ${err.message}`);
  }
}
