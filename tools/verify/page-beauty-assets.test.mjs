import test from 'node:test';
import assert from 'node:assert/strict';
import {mkdtempSync,mkdirSync,writeFileSync,rmSync,readFileSync} from 'node:fs';
import {tmpdir} from 'node:os';
import {join} from 'node:path';
import {createHash} from 'node:crypto';
import {measureBeautyAssets,validateBeautyBudget,readBeautyBaseline} from './page-beauty-assets.mjs';
const ids=['A01','A04','A10','A12','A21'];
function fixture(){const dir=mkdtempSync(join(tmpdir(),'page-beauty-assets-'));mkdirSync(join(dir,'static/chunks'),{recursive:true});writeFileSync(join(dir,'static/chunks/app.js'),'console.log("original");');writeFileSync(join(dir,'static/chunks/app.css'),'body{color:#203c2e}');return dir}
const scripts=()=>Object.fromEntries(ids.map(id=>[id,['/_next/static/chunks/app.js','/_next/static/chunks/app.js']]));
const zero=()=>({css:{gzipBytes:0},initialJS:Object.fromEntries(ids.map(id=>[id,{gzipBytes:0}]))});
const large=()=>Array.from({length:1600},(_,i)=>createHash('sha256').update('original-asset-'+i).digest('hex')).join('');
test('按全部CSS和每路由初始脚本去重计量，相同小资源可通过预算',()=>{const dir=fixture();try{writeFileSync(join(dir,'static/chunks/other.css'),'p{margin:0}');const result=measureBeautyAssets(dir,scripts());assert.equal(result.css.assets.length,2);assert.equal(result.initialJS.A04.assets.length,1);assert(result.css.gzipBytes>0);assert(result.initialJS.A01.gzipBytes>0);assert.equal(validateBeautyBudget(result,zero()),true)}finally{rmSync(dir,{recursive:true})}});
test('真实新增大CSS资产超过10KiB增量时拒绝',()=>{const dir=fixture();try{writeFileSync(join(dir,'static/chunks/large.css'),'/*'+large()+'*/');assert.throws(()=>validateBeautyBudget(measureBeautyAssets(dir,scripts()),zero()),/CSS/)}finally{rmSync(dir,{recursive:true})}});
test('真实初始JS超过8KiB增量时拒绝，遗漏代表路由也拒绝',()=>{const dir=fixture();try{writeFileSync(join(dir,'static/chunks/app.js'),'/*'+large()+'*/');assert.throws(()=>validateBeautyBudget(measureBeautyAssets(dir,scripts()),zero()),/JS/);const missing=scripts();delete missing.A10;assert.throws(()=>measureBeautyAssets(dir,missing),/route|路由/)}finally{rmSync(dir,{recursive:true})}});
test('拒绝静态资源越界和空脚本清单',()=>{const dir=fixture();try{for(const value of ['/_next/static/../../private.js','https://outside.test/app.js','/_next/static/chunks/../app.js'])assert.throws(()=>measureBeautyAssets(dir,{...scripts(),A01:[value]}),/path|路径/);assert.throws(()=>measureBeautyAssets(dir,{...scripts(),A04:[]}),/script|脚本/)}finally{rmSync(dir,{recursive:true})}});
test('T1准确基线固定，不能通过重置基线规避资源门禁',()=>{assert.equal(readBeautyBaseline().sourceCommit,'bf8e02c2193ffca1c6534b1991399cacce62ce12');const dir=fixture();try{const path=join(dir,'baseline.json');const value=JSON.parse(readFileSync(new URL('../../docs/operations/evidence/page-beauty/baseline.json',import.meta.url)));value.css.gzipBytes+=10240;writeFileSync(path,JSON.stringify(value));assert.throws(()=>readBeautyBaseline(path),/baseline|基线/)}finally{rmSync(dir,{recursive:true})}});
