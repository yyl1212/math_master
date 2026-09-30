import { existsSync, realpathSync, readdirSync, lstatSync, readFileSync, mkdirSync, writeFileSync, rmSync } from 'node:fs';
import { resolve, relative, join, isAbsolute, dirname } from 'node:path';
import { createHash } from 'node:crypto';
const hash=b=>createHash('sha256').update(b).digest('hex');
const safe=p=>typeof p==='string'&&p.length>0&&!isAbsolute(p)&&!p.includes('\\')&&!p.split('/').some(s=>s==='..'||s==='.'||!s);
function walk(root,dir='') { return readdirSync(join(root,dir)).sort().flatMap(n=>{if(n==='.DS_Store')return [];const p=dir?dir+'/'+n:n;const s=lstatSync(join(root,p));if(s.isSymbolicLink())throw Error('SOURCE_SYMLINK');return s.isDirectory()?walk(root,p):s.isFile()?[p]:[];}); }
let created;
try {
  const args=process.argv.slice(2), options={};for(let i=0;i<args.length;i+=2){const k=args[i];if(!['--source','--out','--previous'].includes(k)||!args[i+1]||options[k])throw Error('INVALID_ARGUMENT');options[k]=args[i+1];}
  if(!options['--source']||!options['--out'])throw Error('INVALID_ARGUMENT');
  const source=realpathSync(options['--source']),out=resolve(options['--out']);
  if(existsSync(out))throw Error('OUTPUT_EXISTS');
  const parent=realpathSync(dirname(out)),canonicalOut=join(parent,out.split('/').at(-1));
  const rel=relative(source,canonicalOut);if(!rel||(!rel.startsWith('..'+ '/')&&rel!=='..'&&!isAbsolute(rel)))throw Error('OUTPUT_IN_SOURCE');
  const paths=walk(source),captured=new Map(paths.map(p=>[p,readFileSync(join(source,p))]));
  const indices=['Knowledge_JSON_Index.json','Incremental_Knowledge_JSON_Index.json'].filter(p=>captured.has(p));if(!indices.includes('Knowledge_JSON_Index.json'))throw Error('INDEX_REQUIRED');
  const declared=new Map(),primary=new Set();let packageCount=0;
  for(const name of indices){const index=JSON.parse(captured.get(name));for(const pkg of index.packages??[]){packageCount++;const explicit=pkg.primary_knowledge_local_relative_paths??[];for(const p of explicit){if(!safe(p))throw Error('INDEX_PATH_ESCAPE');primary.add(p);}for(const file of pkg.json_files??[]){const p=file.local_relative_path;if(!safe(p))throw Error('INDEX_PATH_ESCAPE');declared.set(p,file.sha256);if(explicit.length===0&&(file.recommended_knowledge_entry||(pkg.primary_knowledge_files??[]).includes(file.filename)))primary.add(p);}}}
  const files=paths.map(path=>({path,sizeBytes:captured.get(path).length,sha256:hash(captured.get(path))}));
  let before=[];if(options['--previous']){const old=JSON.parse(readFileSync(options['--previous']));if(old.schemaVersion!==1||!Array.isArray(old.files))throw Error('INVALID_PREVIOUS');const ids=new Set();for(const f of old.files){if(!safe(f.path)||! /^[a-f0-9]{64}$/.test(f.sha256)||ids.has(f.path))throw Error('INVALID_PREVIOUS');ids.add(f.path);}before=old.files;}
  const oldMap=new Map(before.map(f=>[f.path,f.sha256])),now=new Map(files.map(f=>[f.path,f.sha256]));
  const changes={added:files.filter(f=>!oldMap.has(f.path)).map(f=>f.path),modified:files.filter(f=>oldMap.has(f.path)&&oldMap.get(f.path)!==f.sha256).map(f=>f.path),missing:before.filter(f=>!now.has(f.path)).map(f=>f.path).sort()};
  const sourceIndexMismatches=[...declared].filter(([p,h])=>!now.has(p)||(h&&now.get(p)!==h)).map(([p])=>p).sort();
  const manifest={schemaVersion:1,snapshotId:hash(JSON.stringify(files)),createdAt:new Date().toISOString(),sourceRoot:source,packageCount,primaryFiles:[...primary].sort(),files,changes,sourceIndexMismatches};
  mkdirSync(out);created=out;mkdirSync(join(out,'files'));for(const [p,b] of captured){mkdirSync(dirname(join(out,'files',p)),{recursive:true});writeFileSync(join(out,'files',p),b,{flag:'wx'});}
  const after=walk(source);if(JSON.stringify(after)!==JSON.stringify(paths)||files.some(f=>hash(readFileSync(join(source,f.path)))!==f.sha256))throw Error('SOURCE_CHANGED_RETRY');
  writeFileSync(join(out,'manifest.json'),JSON.stringify(manifest,null,2)+'\n',{flag:'wx'});created=undefined;
  process.stdout.write(JSON.stringify({snapshotId:manifest.snapshotId,fileCount:files.length,packageCount,primaryFileCount:primary.size,changeCounts:Object.fromEntries(Object.entries(changes).map(([k,v])=>[k,v.length])),sourceIndexMismatchCount:sourceIndexMismatches.length})+'\n');
} catch(e){if(created)rmSync(created,{recursive:true,force:true});process.stderr.write(JSON.stringify({error:String(e.message).match(/^[A-Z_]+$/)?.[0]??'SNAPSHOT_FAILED'})+'\n');process.exitCode=1;}
