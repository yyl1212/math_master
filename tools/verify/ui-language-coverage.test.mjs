import test from 'node:test';import assert from 'node:assert/strict';
import {validateUiCoverage} from './ui-language-coverage.mjs';
const entry=file=>({file,module:'public',complete:true,messageKeys:['nav.knowledgeMap'],rawDataAreas:['knowledge.title'],scenarios:['tests/e2e/ui-language-public.spec.ts']});
const policy={pages:['frontend/src/app/page.tsx'],components:['frontend/src/components/site-header.tsx'],keys:['nav.knowledgeMap'],scenarios:['tests/e2e/ui-language-public.spec.ts']};
const manifest=()=>({schemaVersion:1,pages:policy.pages.map(entry),components:policy.components.map(entry)});
test('valid coverage is accepted and a newly added page is rejected',()=>{assert.equal(validateUiCoverage(manifest(),policy),true);assert.throws(()=>validateUiCoverage(manifest(),{...policy,pages:[...policy.pages,'frontend/src/app/new/page.tsx']}),/missing/)});
test('completion requires real scenarios, keys and an explicit data boundary',()=>{for(const field of ['messageKeys','rawDataAreas','scenarios']){const m=manifest();m.pages[0][field]=[];assert.throws(()=>validateUiCoverage(m,policy),/coverage/)}const m=manifest();m.components[0].complete=false;assert.throws(()=>validateUiCoverage(m,policy),/incomplete/)});
test('duplicates, unknown keys and invented scenarios are rejected',()=>{const m=manifest();m.pages.push(m.pages[0]);assert.throws(()=>validateUiCoverage(m,policy),/duplicate/);const k=manifest();k.pages[0].messageKeys=['unknown'];assert.throws(()=>validateUiCoverage(k,policy),/unknown/);const s=manifest();s.pages[0].scenarios=['invented'];assert.throws(()=>validateUiCoverage(s,policy),/unknown/)});

test('only explicitly audited nontext components may omit their own keys',()=>{const m=manifest();m.components[0].messageKeys=[];m.components[0].noOwnText=true;assert.throws(()=>validateUiCoverage(m,policy),/coverage/);assert.equal(validateUiCoverage(m,{...policy,noOwnText:policy.components}),true);m.pages[0].messageKeys=[];m.pages[0].noOwnText=true;assert.throws(()=>validateUiCoverage(m,{...policy,noOwnText:[...policy.components,...policy.pages]}),/coverage/)});
import {readFileSync,readdirSync} from 'node:fs';
import {join,relative} from 'node:path';
const root=new URL('../../',import.meta.url);
function files(folder){return readdirSync(folder,{withFileTypes:true}).flatMap(e=>e.isDirectory()?files(join(folder,e.name)):[join(folder,e.name)]);}
test('repository page and component inventory has verified coverage',()=>{
 const base=root.pathname,read=p=>readFileSync(new URL(p,root),'utf8'),rel=p=>relative(base,p).replaceAll('\\','/');
 const pages=files(join(base,'frontend/src/app')).filter(p=>p.endsWith('/page.tsx')).map(rel);
 const components=['components','features'].flatMap(f=>files(join(base,'frontend/src/'+f))).filter(p=>p.endsWith('.tsx')&&!p.includes('.test.')&&!p.includes('test-fixtures')).map(rel);
 const entries=JSON.parse(read('frontend/src/lib/i18n/messages/en.ts').match(/= ([\s\S]*) as const;/)[1]);
 const scenarios=files(join(base,'tests/e2e')).filter(p=>/ui-language-[a-z]+\.spec\.ts$/.test(p)).map(rel);
 assert.equal(validateUiCoverage(JSON.parse(read('docs/operations/ui-language-coverage.json')),{pages,components,keys:Object.keys(entries),scenarios,noOwnText:['frontend/src/features/learning/learning-account.tsx','frontend/src/features/reading/path-connections.tsx']}),true);
});
