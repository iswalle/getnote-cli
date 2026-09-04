#!/usr/bin/env node
// Install the platform binary bundled in this npm package.
// Installation is local-only; no network request is made here.
'use strict';

const fs = require('node:fs');
const os = require('node:os');
const path = require('node:path');
const { spawnSync } = require('node:child_process');
const { version } = require('../package.json');

const platformNames = { darwin: 'darwin', linux: 'linux', win32: 'windows' };
const archNames = { x64: 'amd64', arm64: 'arm64' };

function getPlatform(platform = os.platform(), arch = os.arch()) {
  const name = platformNames[platform];
  const cpu = archNames[arch];
  if (!name || !cpu) throw new Error(`Unsupported platform: ${platform}/${arch}`);
  return { platform: name, arch: cpu };
}

function getBinaryName(platform) {
  return platform.platform === 'windows' ? 'getnote.exe' : 'getnote';
}

function getPrebuiltPath(platform) {
  return path.join(__dirname, '..', 'prebuilt', `${platform.platform}-${platform.arch}`, getBinaryName(platform));
}

function installFromPrebuilt(platform, binDir) {
  const source = getPrebuiltPath(platform);
  const destination = path.join(binDir, getBinaryName(platform));
  if (!fs.existsSync(source)) {
    throw new Error(`Bundled binary is missing: ${path.relative(path.join(__dirname, '..'), source)}. Install a newer @getnote/cli package.`);
  }
  fs.mkdirSync(binDir, { recursive: true });
  fs.copyFileSync(source, destination);
  if (platform.platform !== 'windows') fs.chmodSync(destination, 0o755);
  console.log(`getnote v${version} installed from the npm package`);
}

function main() {
  const platform = getPlatform();
  const binDir = path.join(__dirname, '..', 'bin');
  const binaryPath = path.join(binDir, getBinaryName(platform));
  if (fs.existsSync(binaryPath)) {
    const result = spawnSync(binaryPath, ['version'], { encoding: 'utf8' });
    if (result.status === 0 && (result.stdout || '').includes(version)) return;
  }
  installFromPrebuilt(platform, binDir);
}

if (require.main === module) {
  try { main(); } catch (error) {
    console.error(`Failed to install getnote: ${error.message}`);
    process.exitCode = 1;
  }
}

module.exports = { getPlatform, getBinaryName, getPrebuiltPath, installFromPrebuilt };
