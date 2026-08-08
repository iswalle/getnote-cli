#!/usr/bin/env node
// Platform-agnostic launcher for getnote binary

'use strict';

const { spawn } = require('child_process');
const path = require('path');
const os = require('os');

const platform = os.platform();
const binaryName = platform === 'win32' ? 'getnote.exe' : 'getnote';
const binaryPath = path.join(__dirname, binaryName);

const child = spawn(binaryPath, process.argv.slice(2), {
  stdio: 'inherit',
  windowsHide: true
});

child.on('exit', (code, signal) => {
  if (signal) {
    process.kill(process.pid, signal);
  } else {
    process.exit(code);
  }
});
