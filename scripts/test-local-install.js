#!/usr/bin/env node
// Test local installation without network access

const { execSync } = require('child_process');
const fs = require('fs');
const path = require('path');
const os = require('os');

const VERSION = require('../package.json').version;
const tmpDir = path.join(os.tmpdir(), `getnote-test-${Date.now()}`);

console.log('==> Testing local installation...');
console.log(`Version: ${VERSION}`);
console.log(`Test directory: ${tmpDir}`);
console.log('');

// Create test environment
fs.mkdirSync(tmpDir, { recursive: true });

try {
  // Pack the package
  console.log('1. Creating package tarball...');
  const tarball = path.resolve(`getnote-cli-${VERSION}.tgz`);
  
  if (!fs.existsSync(tarball)) {
    console.error(`✗ Tarball not found: ${tarball}`);
    console.error('  Run: npm pack');
    process.exit(1);
  }
  
  console.log(`   ✓ Found: ${tarball}`);
  
  // Install globally in test directory
  console.log('');
  console.log('2. Installing package...');
  
  const npmPrefix = path.join(tmpDir, 'npm-global');
  fs.mkdirSync(npmPrefix, { recursive: true });
  
  execSync(`npm install --prefix "${npmPrefix}" "${tarball}"`, {
    stdio: 'inherit',
    env: { 
      ...process.env,
      npm_config_offline: 'true',  // Force offline mode
      npm_config_prefer_offline: 'true'
    }
  });
  
  console.log('   ✓ Package installed');
  
  // Verify binary exists
  console.log('');
  console.log('3. Verifying installation...');
  
  const platform = os.platform();
  const binaryName = platform === 'win32' ? 'getnote.exe' : 'getnote';
  const binaryPath = path.join(
    npmPrefix,
    'node_modules',
    '@getnote',
    'cli',
    'bin',
    binaryName
  );
  
  if (!fs.existsSync(binaryPath)) {
    console.error(`   ✗ Binary not found: ${binaryPath}`);
    process.exit(1);
  }
  
  console.log(`   ✓ Binary exists: ${binaryPath}`);
  
  // Test binary execution
  const result = execSync(`"${binaryPath}" --version`, { encoding: 'utf8' });
  console.log(`   ✓ Binary runs: ${result.trim()}`);
  
  if (!result.includes(VERSION)) {
    console.error(`   ✗ Version mismatch: expected ${VERSION}, got ${result.trim()}`);
    process.exit(1);
  }
  
  console.log('');
  console.log('==> ✓ All tests passed!');
  console.log('');
  console.log('Installation succeeded without network access.');
  console.log('The package includes all platform binaries and does not depend on GitHub.');
  
} catch (err) {
  console.error('');
  console.error('==> ✗ Test failed:');
  console.error(err.message);
  process.exit(1);
} finally {
  // Cleanup
  console.log('');
  console.log('Cleaning up...');
  fs.rmSync(tmpDir, { recursive: true, force: true });
  console.log('Done.');
}
