#!/usr/bin/env node

import { execSync } from 'child_process';
import fs from 'fs';
import path from 'path';
import { fileURLToPath } from 'url';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

// Ensure the output directory exists
const outputDir = path.join(__dirname, 'src', 'gen', 'proto');
if (!fs.existsSync(outputDir)) {
  fs.mkdirSync(outputDir, { recursive: true });
}

// Path to the proto file
const protoFile = path.join(__dirname, '..', 'server', 'proto', 'ironbird.proto');

// Run protoc with the ES and Connect plugins
try {
  console.log('Generating TypeScript code from proto files...');
  
  const command = `
    protoc \
      --es_out=src/gen/proto \
      --es_opt=target=ts \
      --connect-es_out=src/gen/proto \
      --connect-es_opt=target=ts \
      -I=../server/proto \
      ../server/proto/ironbird.proto
  `;
  
  execSync(command, { cwd: __dirname, stdio: 'inherit' });
  console.log('TypeScript code generation completed successfully!');
} catch (error) {
  console.error('Error generating TypeScript code:', error.message);
  process.exit(1);
}
