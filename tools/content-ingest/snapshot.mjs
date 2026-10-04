import { readIndexDeclarations } from './source-index.mjs';
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
  const audit=readIndexDeclarations(captured),primary=new Set(audit.primaryFiles),packageCount=audit.packageCount;
  const files=paths.map(path=>({path,sizeBytes:captured.get(path).length,sha256:hash(captured.get(path))}));
  let before=[];if(options['--previous']){const old=JSON.parse(readFileSync(options['--previous']));if(old.schemaVersion!==1||!Array.isArray(old.files))throw Error('INVALID_PREVIOUS');const ids=new Set();for(const f of old.files){if(!safe(f.path)||! /^[a-f0-9]{64}$/.test(f.sha256)||ids.has(f.path))throw Error('INVALID_PREVIOUS');ids.add(f.path);}before=old.files;}
  const oldMap=new Map(before.map(f=>[f.path,f.sha256])),now=new Map(files.map(f=>[f.path,f.sha256]));
  const changes={added:files.filter(f=>!oldMap.has(f.path)).map(f=>f.path),modified:files.filter(f=>oldMap.has(f.path)&&oldMap.get(f.path)!==f.sha256).map(f=>f.path),missing:before.filter(f=>!now.has(f.path)).map(f=>f.path).sort()};
  const sourceIndexMismatches=audit.declarations.filter(({path,claims})=>!now.has(path)||claims.some(c=>c.sha256&&now.get(path)!==c.sha256)).map(d=>d.path).sort();
  const manifest={schemaVersion:1,snapshotId:hash(JSON.stringify(files)),createdAt:new Date().toISOString(),sourceRoot:source,packageCount,primaryFiles:[...primary].sort(),files,changes,sourceIndexMismatches};
  mkdirSync(out,{mode:0o700});created=out;mkdirSync(join(out,'files'),{mode:0o700});for(const [p,b] of captured){mkdirSync(dirname(join(out,'files',p)),{recursive:true,mode:0o700});writeFileSync(join(out,'files',p),b,{flag:'wx',mode:0o600});}
  const after=walk(source);if(JSON.stringify(after)!==JSON.stringify(paths)||files.some(f=>hash(readFileSync(join(source,f.path)))!==f.sha256))throw Error('SOURCE_CHANGED_RETRY');
  writeFileSync(join(out,'manifest.json'),JSON.stringify(manifest,null,2)+'\n',{flag:'wx',mode:0o600});created=undefined;
  process.stdout.write(JSON.stringify({snapshotId:manifest.snapshotId,fileCount:files.length,packageCount,primaryFileCount:primary.size,changeCounts:Object.fromEntries(Object.entries(changes).map(([k,v])=>[k,v.length])),sourceIndexMismatchCount:sourceIndexMismatches.length})+'\n');
} catch(e){if(created)rmSync(created,{recursive:true,force:true});process.stderr.write(JSON.stringify({error:String(e.message).match(/^[A-Z_]+$/)?.[0]??'SNAPSHOT_FAILED'})+'\n');process.exitCode=1;}
