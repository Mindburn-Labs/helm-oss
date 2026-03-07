import { readFile } from 'node:fs/promises';
import { join } from 'node:path';

import { verifyBundle } from '../packages/mindburn-helm-cli/src/verify.ts';

interface ExpectedFixture {
  bundle_root: string;
  merkle_root: string;
  expected_verdict: 'PASS' | 'FAIL';
}

async function main(): Promise<void> {
  const fixtureDir = join(import.meta.dirname, '..', 'fixtures', 'minimal');
  const expectedPath = join(fixtureDir, 'EXPECTED.json');
  const expected = JSON.parse(await readFile(expectedPath, 'utf8')) as ExpectedFixture;
  const result = await verifyBundle(fixtureDir, 'L2');

  const mismatches: string[] = [];

  if (result.verdict !== expected.expected_verdict) {
    mismatches.push(
      `verdict mismatch: expected ${expected.expected_verdict}, got ${result.verdict}`,
    );
  }
  if (result.roots.manifest_root_hash !== expected.bundle_root) {
    mismatches.push(
      `bundle root mismatch: expected ${expected.bundle_root}, got ${result.roots.manifest_root_hash}`,
    );
  }
  if (result.roots.merkle_root !== expected.merkle_root) {
    mismatches.push(
      `merkle root mismatch: expected ${expected.merkle_root}, got ${result.roots.merkle_root}`,
    );
  }

  if (mismatches.length > 0) {
    throw new Error(mismatches.join('\n'));
  }

  process.stdout.write(
    [
      'Fixture roots verified:',
      `  verdict: ${result.verdict}`,
      `  bundle_root: ${result.roots.manifest_root_hash}`,
      `  merkle_root: ${result.roots.merkle_root}`,
    ].join('\n') + '\n',
  );
}

await main();
