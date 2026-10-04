import { isAbsolute } from 'node:path';
export const indexNames = ['Knowledge_JSON_Index.json', 'Incremental_Knowledge_JSON_Index.json', 'MSC2020_Incremental_Knowledge_JSON_Index.json'];
export function safeRelativePath(path) {
  return typeof path === 'string' && path.length > 0 && !isAbsolute(path) && !path.includes('\\') && !path.split('/').some(part => part === '..' || part === '.' || !part);
}
export function readIndexDeclarations(captured) {
  const indices = indexNames.filter(name => captured.has(name));
  if (!indices.includes(indexNames[0])) throw Error('INDEX_REQUIRED');
  const declarations = new Map(), primary = new Set();
  let packageCount = 0;
  for (const name of indices) {
    const index = JSON.parse(captured.get(name));
    for (const pkg of index.packages ?? []) {
      packageCount++;
      const explicit = pkg.primary_knowledge_local_relative_paths ?? [];
      for (const path of explicit) {
        if (!safeRelativePath(path)) throw Error('INDEX_PATH_ESCAPE');
        primary.add(path);
      }
      for (const file of pkg.json_files ?? []) {
        const path = file.local_relative_path;
        if (!safeRelativePath(path)) throw Error('INDEX_PATH_ESCAPE');
        const claims = declarations.get(path) ?? [];
        const claim = { index: name, sha256: file.sha256 ?? null };
        if (!claims.some(c => c.index === claim.index && c.sha256 === claim.sha256)) claims.push(claim);
        declarations.set(path, claims);
        if (explicit.length === 0 && (file.recommended_knowledge_entry || (pkg.primary_knowledge_files ?? []).includes(file.filename))) primary.add(path);
      }
    }
  }
  const issues = [];
  const entries = [...declarations].sort(([a], [b]) => a < b ? -1 : a > b ? 1 : 0).map(([path, claims]) => {
    if (claims.some(c => !c.sha256)) issues.push({ code: 'INDEX_DIGEST_MISSING', path });
    if (claims.some(c => c.sha256 && (typeof c.sha256 !== 'string' || !/^[a-f0-9]{64}$/.test(c.sha256)))) issues.push({ code: 'INDEX_DIGEST_INVALID', path });
    if (new Set(claims.filter(c => c.sha256).map(c => c.sha256)).size > 1) issues.push({ code: 'INDEX_DIGEST_CONFLICT', path });
    return { path, claims };
  });
  return { indices, packageCount, primaryFiles: [...primary].sort(), declarations: entries, issues };
}
