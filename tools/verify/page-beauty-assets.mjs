import assert from 'node:assert/strict';
import {readFileSync,readdirSync,lstatSync,writeFileSync} from 'node:fs';
import {resolve,join,relative} from 'node:path';
import {fileURLToPath} from 'node:url';
import {createHash} from 'node:crypto';
import {gzipSync} from 'node:zlib';
const root=fileURLToPath(new URL('../../',import.meta.url));
const baselinePath=join(root,'docs/operations/evidence/page-beauty/baseline.json');
const baselineSha='c7a71079ef43492d86d0206d7f2f1627ef343d67c3e53625b5f0c688aee8c61a';
export const beautyRouteIds=['A01','A04','A10','A12','A21'];
const compressed=bytes=>gzipSync(bytes,{level:9}).length;
export function readBeautyBaseline(path=baselinePath){
 const raw=readFileSync(path);assert.equal(createHash('sha256').update(raw).digest('hex'),baselineSha,'T1 baseline 基线不可重置');
 const value=JSON.parse(raw);assert.equal(value.sourceCommit,'bf8e02c2193ffca1c6534b1991399cacce62ce12');assert.equal(value.measurement.gzipLevel,9);return value;
}
export function measureBeautyAssets(buildDir,routeScripts){
 buildDir=resolve(buildDir);assert.deepEqual(Object.keys(routeScripts).sort(),[...beautyRouteIds].sort(),'required route 路由清单');
 const css=[];
 function walk(folder){for(const entry of readdirSync(folder,{withFileTypes:true})){const path=join(folder,entry.name);assert(!entry.isSymbolicLink(),'asset path 路径不可为符号链接');if(entry.isDirectory())walk(path);else if(entry.name.endsWith('.css'))css.push({path:relative(buildDir,path).replaceAll('\\','/'),gzipBytes:compressed(readFileSync(path))})}}
 walk(join(buildDir,'static'));assert(css.length>0,'CSS assets missing');css.sort((a,b)=>a.path.localeCompare(b.path));
 const initialJS={};
 for(const id of beautyRouteIds){
  const scripts=routeScripts[id];assert(Array.isArray(scripts)&&scripts.length>0,'initial script 脚本清单不能为空');
  const assets=[...new Set(scripts)].map(path=>{
   assert(typeof path==='string'&&/^\/_next\/static\/[A-Za-z0-9_./-]+\.js$/.test(path)&&!path.split('/').some(part=>part==='..'||part==='.'),'unsafe asset path 路径');
   const file=join(buildDir,path.slice('/_next/'.length));assert(lstatSync(file).isFile()&&!lstatSync(file).isSymbolicLink(),'asset path 路径不是普通文件');
   return {path,gzipBytes:compressed(readFileSync(file))};
  }).sort((a,b)=>a.path.localeCompare(b.path));
  initialJS[id]={assets,gzipBytes:assets.reduce((n,a)=>n+a.gzipBytes,0)};
 }
 return {measurement:{gzipLevel:9,css:'all unique .next/static/**/*.css',js:'unique initial HTML script[src] at same authenticated representative routes'},css:{assets:css,gzipBytes:css.reduce((n,a)=>n+a.gzipBytes,0)},initialJS};
}
export function validateBeautyBudget(current,baseline){
 assert(Number.isSafeInteger(current.css.gzipBytes)&&current.css.gzipBytes>0,'CSS measurement');
 assert(current.css.gzipBytes-baseline.css.gzipBytes<=10*1024,'CSS gzip增量超过10KiB');
 for(const id of beautyRouteIds){assert(Number.isSafeInteger(current.initialJS[id]?.gzipBytes)&&current.initialJS[id].gzipBytes>0,'JS required route '+id);assert(current.initialJS[id].gzipBytes-baseline.initialJS[id].gzipBytes<=8*1024,'JS gzip增量超过8KiB: '+id)}
 return true;
}
if(process.argv[1]&&resolve(process.argv[1])===fileURLToPath(import.meta.url)){
 const args=process.argv.slice(2),at=args.indexOf('--capture');assert(at>=0&&args[at+1],'--capture required');const capture=resolve(args[at+1]),baseline=readBeautyBaseline();
 const scripts=Object.fromEntries(beautyRouteIds.map(id=>{const data=JSON.parse(readFileSync(join(capture,id+'-en-1280-assets.json'),'utf8'));assert.equal(data.route,baseline.initialJS[id].route,'same representative route '+id);return [id,data.assets.map(asset=>asset.path)]}));
 const current=measureBeautyAssets(join(root,'frontend/.next'),scripts);validateBeautyBudget(current,baseline);
 const report={baselineCommit:baseline.sourceCommit,baselineSha,gzipLevel:9,css:{before:baseline.css.gzipBytes,after:current.css.gzipBytes,delta:current.css.gzipBytes-baseline.css.gzipBytes,budget:10240},initialJS:Object.fromEntries(beautyRouteIds.map(id=>[id,{before:baseline.initialJS[id].gzipBytes,after:current.initialJS[id].gzipBytes,delta:current.initialJS[id].gzipBytes-baseline.initialJS[id].gzipBytes,budget:8192}])),assets:current};
 writeFileSync(join(capture,'assets-report.json'),JSON.stringify(report,null,2)+'\n');console.log(JSON.stringify({ok:true,css:report.css,initialJS:report.initialJS}));
}
