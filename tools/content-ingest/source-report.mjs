import { readFileSync, lstatSync, realpathSync, existsSync, writeFileSync } from 'node:fs';
import { resolve, join, relative, dirname, isAbsolute } from 'node:path';
import { fileURLToPath } from 'node:url';
import { createHash } from 'node:crypto';
import { indexNames, safeRelativePath, readIndexDeclarations } from './source-index.mjs';
const sha256 = bytes => createHash('sha256').update(bytes).digest('hex');
const contains = (root, path) => { const r = relative(root, path); return !r || (!isAbsolute(r) && r !== '..' && !r.startsWith('../')); };
function fixedFile(root, path) {
  let current = root;
  for (const part of ['', ...path.split('/')]) {
    if (part) current = join(current, part);
    const stat = lstatSync(current);
    if (stat.isSymbolicLink()) throw Error('SNAPSHOT_SYMLINK');
  }
  if (!lstatSync(current).isFile()) throw Error('INVALID_SNAPSHOT');
  return readFileSync(current);
}
export function buildSourceReport(snapshotDir, selectedFiles) {
  if (!Array.isArray(selectedFiles) || !selectedFiles.length || selectedFiles.length > 1000 || new Set(selectedFiles).size !== selectedFiles.length || selectedFiles.some(p => !safeRelativePath(p))) throw Error('INVALID_SELECTION');
  const root = resolve(snapshotDir);
  const manifest = JSON.parse(fixedFile(root, 'manifest.json'));
  if (manifest.schemaVersion !== 1 || !Array.isArray(manifest.files) || !manifest.files.length || manifest.files.length > 10000 || sha256(JSON.stringify(manifest.files)) !== manifest.snapshotId) throw Error('INVALID_SNAPSHOT');
  const selected = new Set(selectedFiles), captured = new Map(), files = new Map();
  for (const entry of manifest.files) {
    if (!safeRelativePath(entry.path) || files.has(entry.path) || !Number.isSafeInteger(entry.sizeBytes) || entry.sizeBytes < 0 || !/^[a-f0-9]{64}$/.test(entry.sha256)) throw Error('INVALID_SNAPSHOT');
    const bytes = fixedFile(join(root, 'files'), entry.path);
    if (bytes.length !== entry.sizeBytes || sha256(bytes) !== entry.sha256) throw Error('SNAPSHOT_BYTES_MISMATCH');
    files.set(entry.path, entry);
    if (selected.has(entry.path) || indexNames.includes(entry.path)) captured.set(entry.path, bytes);
  }
  const audit = readIndexDeclarations(captured), declarations = new Map(audit.declarations.map(x => [x.path, x.claims]));
  const issues = [];
  const issue = (code, path) => issues.push({ code, path, indexClaims: declarations.get(path) ?? [], actualSha256: files.get(path)?.sha256 ?? null, blocksSelected: selected.has(path) });
  for (const item of audit.issues) issue(item.code, item.path);
  for (const {path, claims} of audit.declarations) {
    if (!files.has(path)) issue('INDEX_FILE_MISSING', path);
    else if (claims.some(c => c.sha256 && c.sha256 !== files.get(path).sha256)) issue('INDEX_DIGEST_MISMATCH', path);
  }
  const selectedEntries = [];
  for (const path of selectedFiles) {
    if (!files.has(path)) { issue('SELECTED_FILE_MISSING', path); continue; }
    if (!declarations.has(path)) issue('SELECTED_FILE_UNDECLARED', path);
    let corpus;
    try { corpus = JSON.parse(captured.get(path)); } catch { issue('CORPUS_JSON_INVALID', path); continue; }
    const recordIds = [], ids = new Set();
    if (!corpus || !corpus.schema_version || typeof corpus.dataset_id !== 'string' || !corpus.dataset_id || !Array.isArray(corpus.knowledge_points) || !corpus.knowledge_points.length) issue('CORPUS_SCHEMA_INVALID', path);
    else for (const record of corpus.knowledge_points) {
      if (typeof record?.id !== 'string' || !record.id.trim() || record.id.includes('\0')) { issue('RECORD_ID_MISSING', path); continue; }
      if (ids.has(record.id)) issue('RECORD_ID_CONFLICT', path);
      else { ids.add(record.id); recordIds.push(record.id); }
    }
    selectedEntries.push({path,sha256:files.get(path).sha256,datasetId:typeof corpus?.dataset_id === 'string' ? corpus.dataset_id : '',recordIds});
  }
  return {schemaVersion:1,policyVersion:1,snapshotId:manifest.snapshotId,selectedFiles:selectedEntries,issues,ready:!issues.some(x=>x.blocksSelected),publicationApproved:false};
}
function cli() {
  try {
    const options = {}, args = process.argv.slice(2);
    for (let i = 0; i < args.length; i += 2) {
      if (!['--snapshot','--selected','--out'].includes(args[i]) || !args[i+1] || options[args[i]]) throw Error('INVALID_ARGUMENT');
      options[args[i]] = args[i+1];
    }
    if (!options['--snapshot'] || !options['--selected'] || !options['--out']) throw Error('INVALID_ARGUMENT');
    const root = resolve(options['--snapshot']);
    const report = buildSourceReport(root, JSON.parse(readFileSync(options['--selected'])));
    const out = resolve(options['--out']);
    if (existsSync(out)) throw Error('OUTPUT_EXISTS');
    const parent = realpathSync(dirname(out)), target = join(parent, out.split('/').at(-1));
    const manifest = JSON.parse(fixedFile(root, 'manifest.json'));
    if (typeof manifest.sourceRoot !== 'string' || contains(realpathSync(manifest.sourceRoot), target)) throw Error('OUTPUT_IN_SOURCE');
    if ((lstatSync(parent).mode & 0o777) !== 0o700) throw Error('OUTPUT_NOT_PRIVATE');
    writeFileSync(target, JSON.stringify(report,null,2)+'\n', {flag:'wx',mode:0o600});
    process.stdout.write(JSON.stringify({snapshotId:report.snapshotId,selectedFileCount:report.selectedFiles.length,issueCount:report.issues.length,ready:report.ready,publicationApproved:false})+'\n');
  } catch(error) { process.stderr.write(JSON.stringify({error:/^[A-Z_]+$/.test(error.message) ? error.message : 'SOURCE_REPORT_FAILED'})+'\n'); process.exitCode = 1; }
}
if (process.argv[1] && resolve(process.argv[1]) === fileURLToPath(import.meta.url)) cli();
