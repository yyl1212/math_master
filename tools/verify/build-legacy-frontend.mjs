import {readFile,writeFile,mkdir,cp,rm} from 'node:fs/promises';
import {resolve,relative,basename,join} from 'node:path';
import {createHash} from 'node:crypto';
import {spawnSync} from 'node:child_process';
import {constants} from 'node:fs';
const root=resolve(import.meta.dirname,'../..'),fixture=join(root,'tests/fixtures/legacy-frontend');
const target=join(root,'.superpowers/sdd/2026-10-11-site-streamline/legacy/frontend');
const manifest=JSON.parse(await readFile(join(fixture,'manifest.json'),'utf8'));
if(manifest.baseCommit!=='06d04a2eb20cf2a56cfec57283224583659c2ffd')throw new Error('Unexpected legacy base');
for(const[path,sha]of Object.entries(manifest.files)){if(path.includes('..')||path.startsWith('/')||!path.startsWith('src/'))throw new Error('Unsafe fixture path');const data=await readFile(join(fixture,path));if(createHash('sha256').update(data).digest('hex')!==sha)throw new Error('Legacy fixture integrity mismatch: '+path);}
await rm(target,{recursive:true,force:true});await mkdir(target,{recursive:true});
await cp(join(root,'frontend'),target,{recursive:true,filter:path=>!['node_modules','.next','tsconfig.tsbuildinfo'].some(part=>relative(join(root,'frontend'),path).split('/').includes(part))&&!basename(path).startsWith('.env')});
await cp(join(root,'content'),join(target,'../content'),{recursive:true});
await cp(join(root,'backend/internal/content/testdata'),join(target,'../backend/internal/content/testdata'),{recursive:true});
await cp(join(root,'frontend/node_modules'),join(target,'node_modules'),{recursive:true,mode:constants.COPYFILE_FICLONE,verbatimSymlinks:true});
for(const path of Object.keys(manifest.files)){const destination=join(target,path);await mkdir(resolve(destination,'..'),{recursive:true});await cp(join(fixture,path),destination);}
await writeFile(join(target,'compatibility-build.json'),JSON.stringify({kind:'legacy-test-only',baseCommit:manifest.baseCommit,fixtureFiles:Object.keys(manifest.files).length}));
const result=spawnSync(process.execPath,[join(root,'tools/verify/run.mjs'),'--cwd',target,'--','npm','run','build'],{cwd:root,env:process.env,stdio:'inherit'});
if(result.status!==0)process.exit(result.status??1);
console.log(JSON.stringify({ok:true,kind:'legacy-test-only',path:target,fixtureFiles:Object.keys(manifest.files).length}));
