#!/bin/bash

set -euo pipefail

cd /app/e2e
npm install
npm rebuild
cd /app
EXECUTABLE=${EXECUTABLE} npx --prefix e2e vitest run --config e2e/vitest.config.ts
