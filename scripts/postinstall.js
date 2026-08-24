#!/usr/bin/env node
// postinstall.js — downloads the getnote binary for the current platform.

'use strict';

const fs = require('fs');
const path = require('path');
const os = require('os');
const https = require('https');
const crypto = require('crypto');
const { spawnSync } = require('child_process');

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

function getWindowsExtractArgs(archivePath, destinationPath) {
  return [
    '-NoProfile',
    '-Command',
    '& { Expand-Archive -LiteralPath $args[0] -DestinationPath $args[1] -Force }',
    archivePath,
    destinationPath,
  ];
}

async function installArchive({
  platform,
  binDir,
  binaryName,
  binaryPath,
  url,
  tmpFile,
  downloadFn = download,
  verifyChecksumFn = verifyChecksum,
  runFn = run,
  chmodFn = fs.chmodSync,
  unlinkFn = fs.unlinkSync,
}) {
  try {
    await downloadFn(url, tmpFile);
    await verifyChecksumFn(url, path.basename(url), tmpFile);
    if (platform.platform === 'windows') {
      runFn('powershell', getWindowsExtractArgs(tmpFile, binDir));
    } else {
      runFn('tar', ['-xzf', tmpFile, '-C', binDir, binaryName]);
    }

    chmodFn(binaryPath, 0o755);
    console.log(`getnote installed at ${binaryPath}`);
  } finally {
    try { unlinkFn(tmpFile); } catch (_) {}
  }
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
      const result = spawnSync(binaryPath, ['version'], { encoding: 'utf8' });
      const fallback = result.status === 0 ? result : spawnSync(binaryPath, ['--version'], { encoding: 'utf8' });
      const out = (fallback.stdout || '').trim();
      if (out.includes(VERSION)) {
        console.log(`getnote v${VERSION} already installed, skipping download.`);
        return;
      }
    } catch (_) { /* binary exists but can't run (wrong arch etc.), re-download */ }
  }

  console.log(`Downloading getnote v${VERSION} for ${platform.platform}/${platform.arch}...`);
  console.log(`URL: ${url}`);

  fs.mkdirSync(binDir, { recursive: true });

  await installArchive({ platform, binDir, binaryName, binaryPath, url, tmpFile });
}

function run(command, args) {
  const result = spawnSync(command, args, { stdio: 'inherit' });
  if (result.error) throw result.error;
  if (result.status !== 0) throw new Error(`${command} exited with status ${result.status}`);
}

function download(url, destination, redirects = 0) {
  if (redirects > 5) return Promise.reject(new Error('Too many redirects'));
  return new Promise((resolve, reject) => {
    const request = https.get(url, { headers: { 'User-Agent': '@getnote/cli installer' } }, response => {
      if (response.statusCode >= 300 && response.statusCode < 400 && response.headers.location) {
        response.resume();
        return resolve(download(new URL(response.headers.location, url).toString(), destination, redirects + 1));
      }
      if (response.statusCode !== 200) {
        response.resume();
        return reject(new Error(`HTTP ${response.statusCode}: ${url}`));
      }
      const output = fs.createWriteStream(destination, { mode: 0o600 });
      response.pipe(output);
      output.on('finish', () => output.close(resolve));
      output.on('error', reject);
    });
    request.on('error', reject);
  });
}

async function verifyChecksum(assetURL, assetName, archivePath) {
  const checksumPath = `${archivePath}.checksums`;
  try {
    await download(new URL('checksums.txt', assetURL).toString(), checksumPath);
    const line = fs.readFileSync(checksumPath, 'utf8').split(/\r?\n/).find(value => {
      const fields = value.trim().split(/\s+/);
      return fields.length === 2 && fields[1].replace(/^\*/, '') === assetName;
    });
    if (!line) throw new Error(`Checksum for ${assetName} is missing`);
    const expected = line.trim().split(/\s+/)[0].toLowerCase();
    if (!/^[a-f0-9]{64}$/.test(expected)) throw new Error(`Checksum for ${assetName} is invalid`);
    const actual = crypto.createHash('sha256').update(fs.readFileSync(archivePath)).digest('hex');
    if (actual !== expected) throw new Error(`Checksum mismatch for ${assetName}`);
  } finally {
    try { fs.unlinkSync(checksumPath); } catch (_) {}
  }
}

if (require.main === module) {
  main().catch(err => {
    console.error('Failed to install getnote:', err.message);
    process.exitCode = 1;
  });
}

module.exports = { getWindowsExtractArgs, installArchive };
