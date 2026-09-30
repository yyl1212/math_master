import test from 'node:test';
import assert from 'node:assert/strict';
import { spawnSync } from 'node:child_process';
import { fileURLToPath } from 'node:url';
const runner = fileURLToPath(new URL('./run.mjs', import.meta.url));
const run = (...args) => spawnSync(process.execPath,[runner,...args],{encoding:'utf8',timeout:10000});
test('keeps success and failure exit codes', () => {
  assert.equal(run('--',process.execPath,'-e','process.exit(0)').status,0);
  assert.equal(run('--',process.execPath,'-e','process.exit(7)').status,7);
});
test('terminates commands on timeout', () => {
  assert.equal(run('--timeout-ms','50','--',process.execPath,'-e','setInterval(()=>{},1000)').status,124);
});
test('rejects timeouts above the test ceiling', () => {
  assert.equal(run('--timeout-ms','540001','--',process.execPath,'-e','process.exit(0)').status,2);
});
