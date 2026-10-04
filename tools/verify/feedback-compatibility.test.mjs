import test from 'node:test';
import assert from 'node:assert/strict';
import { readFileSync, readdirSync } from 'node:fs';
import { createHash } from 'node:crypto';
import {inverseApprovedCorrectionChanges} from './correction-compatibility.mjs';
const root = new URL('../../', import.meta.url);
const read = p => readFileSync(new URL(p, root), 'utf8');
const baseline = JSON.parse(read('api/feedback-compatibility-baseline.json'));
const api = JSON.parse(read('api/openapi.yaml'));
const canonical = v => Array.isArray(v) ? v.map(canonical) : v && typeof v === 'object' ? Object.fromEntries(Object.keys(v).sort().map(k => [k, canonical(v[k])])) : v;
const digest = v => createHash('sha256').update(JSON.stringify(canonical(v))).digest('hex');
function compare(source) {
  const actual = inverseApprovedCorrectionChanges(source);
  for (const [section, entries] of Object.entries(baseline.sections)) {
    const values = section === 'paths' ? actual.paths : actual.components[section];
    for (const [key, sha] of Object.entries(entries)) assert.equal(digest(values[key]), sha, section + ': ' + key);
  }
}
function compareFiles(reader = read) {
  for (const [p, sha] of Object.entries(baseline.files)) assert.equal(createHash('sha256').update(reader(p)).digest('hex'), sha, p);
}
test('approved master preserves all 75 paths, 192 schemas, 32 responses and three security schemes', () => {
  assert.equal(baseline.baseCommit, '8400ee56d59fd13ecf23f83d89d7685027c93c2d');
  assert.deepEqual(baseline.counts, { paths: 75, schemas: 192, responses: 32, securitySchemes: 3 });
  for (const [section, count] of Object.entries(baseline.counts)) assert.equal(Object.keys(baseline.sections[section]).length, count);
  compare(api);
});
test('a changed original Learning field is detected', () => {
  const changed = structuredClone(api);
  changed.components.schemas.LearningSubmitInput.properties.answers.minItems = 4;
  assert.throws(() => compare(changed), /LearningSubmitInput/);
});
test('all original migrations 00001 through 00006 retain their exact bytes', () => {
  assert.equal(Object.keys(baseline.files).length, 6);
  compareFiles();
});
test('changing an original migration is detected', () => {
  const p = Object.keys(baseline.files)[0];
  assert.throws(() => compareFiles(name => read(name) + (name === p ? '\n-- incompatible change\n' : '')), /00001_content_foundation/);
});
test('original mathematical digest purposes retain their exact sets', () => {
  assert.deepEqual(Object.keys(baseline.mathematicalPurposes).sort(), ['assessment', 'content', 'learning', 'publication', 'question'].map(p => 'backend/internal/' + p + '/'));
  for (const [p, names] of Object.entries(baseline.mathematicalPurposes)) {
    const source = readdirSync(new URL(p, root)).filter(f => f.endsWith('.go') && !f.endsWith('_test.go')).map(f => read(p + f)).join('\n');
    const actual = [...new Set([...source.matchAll(/"([a-z-]+-v\d+)"/g)].map(m => m[1]))].sort();
    assert.deepEqual(actual, names, p);
  }
});
