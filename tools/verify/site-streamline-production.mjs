import {readFileSync,readdirSync,existsSync} from 'node:fs';
import {resolve,dirname,relative} from 'node:path';
const root=resolve(import.meta.dirname,'../..'),src=resolve(root,'frontend/src');
const files=folder=>readdirSync(folder,{withFileTypes:true}).flatMap(entry=>entry.isDirectory()?files(resolve(folder,entry.name)):[resolve(folder,entry.name)]);
const starts=files(resolve(src,'app')).filter(file=>file.endsWith('/page.tsx')||file.endsWith('/layout.tsx'));
const denied=/\/features\/(learning|question|correction|notification)\/|\/features\/content\/(?!page-access|content-state)|\/features\/study\/(legacy-archive|topic-area|retired-module|pages)\./;
const visited=new Set();
function visit(file){if(visited.has(file))return;visited.add(file);if(denied.test(file))throw new Error('Legacy business enters production graph: '+relative(root,file));const text=readFileSync(file,'utf8');const imports=[...text.matchAll(/(?:import\s+(?!type\b)[\s\S]*?\sfrom\s*|import\s*\(|export\s+[^;]*?\sfrom\s*)['"]([^'"]+)['"]/g)].map(match=>match[1]);for(const specifier of imports){let base;if(specifier.startsWith('@/'))base=resolve(src,specifier.slice(2));else if(specifier.startsWith('.'))base=resolve(dirname(file),specifier);else continue;const next=[base+'.tsx',base+'.ts',base+'/index.tsx',base+'/index.ts'].find(existsSync);if(next)visit(next)}}
for(const file of starts)visit(file);
const manifest=resolve(root,'frontend/.next/server/app-paths-manifest.json');if(!existsSync(manifest))throw new Error('Production build missing');
const trace=resolve(root,'frontend/.next/standalone/compatibility-build.json');if(existsSync(trace))throw new Error('Compatibility output cannot be production');
for(const file of files(resolve(root,'frontend/.next/server')).filter(path=>path.endsWith('client-reference-manifest.js'))){if(denied.test(readFileSync(file,'utf8').replaceAll('\\\\','/')))throw new Error('Legacy client references in production: '+relative(root,file));}
console.log(JSON.stringify({ok:true,productionEntries:starts.length,reachableModules:visited.size,legacyBusinessModules:0}));
