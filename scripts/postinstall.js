#!/usr/bin/env node
// postinstall.js — downloads the getnote binary for the current platform.

'use strict';

const https = require('https');
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

  const platformMap = {
    darwin: 'darwin',
    linux: 'linux',
    win32: 'windows',
  };
  const archMap = {
    x64: 'amd64',
    arm64: 'arm64',
  };

  const p = platformMap[platform];
  const a = archMap[arch];
  if (!p || !a) {
    throw new Error(`Unsupported platform: ${platform}/${arch}`);
  }
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

async function download(url, dest) {
  return new Promise((resolve, reject) => {
    const file = fs.createWriteStream(dest);
    const get = (u) => {
      https.get(u, (res) => {
        if (res.statusCode === 301 || res.statusCode === 302) {
          file.close();
          get(res.headers.location);
          return;
        }
        if (res.statusCode !== 200) {
          reject(new Error(`Download failed: ${res.statusCode} ${u}`));
          return;
        }
        res.pipe(file);
        file.on('finish', () => file.close(resolve));
      }).on('error', (err) => {
        fs.unlink(dest, () => {});
        reject(err);
      });
    };
    get(url);
  });
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
    await download(url, tmpFile);

    if (platform.platform === 'windows') {
      execSync(`unzip -o "${tmpFile}" "${binaryName}" -d "${binDir}"`);
    } else {
      execSync(`tar -xzf "${tmpFile}" -C "${binDir}" "${binaryName}"`);
    }

    fs.chmodSync(binaryPath, 0o755);
    console.log(`getnote installed at ${binaryPath}`);
  } finally {
    try { fs.unlinkSync(tmpFile); } catch (_) {}
  }
}

main().catch((err) => {
  console.error('Failed to install getnote:', err.message);
  process.exit(1);
});
