import assert from'node:assert/strict';import{createHash}from'node:crypto';import{readFileSync,readdirSync,lstatSync}from'node:fs';import{join,resolve,relative}from'node:path';import{fileURLToPath}from'node:url';
const baselinePath='api/topic-learning-compatibility-baseline.json';
export const TOPIC_COMPATIBILITY_POLICY_SHA='782c4b1fba612a842c4184e9c8cce10e6218c9e18bc805d40755698b773c0ed2';
const sha=bytes=>createHash('sha256').update(bytes).digest('hex');
const canonical=value=>Array.isArray(value)?value.map(canonical):value&&typeof value==='object'?Object.fromEntries(Object.keys(value).sort().map(k=>[k,canonical(value[k])])):value;
const digest=value=>sha(Buffer.from(JSON.stringify(canonical(value))));
function policy(reader){const raw=reader(baselinePath);assert.equal(sha(raw),TOPIC_COMPATIBILITY_POLICY_SHA,'topic compatibility baseline policy');const p=JSON.parse(raw);assert.equal(p.schemaVersion,1);assert.equal(p.baseCommit,'bca91cc98d75af53963789f28cdda5ca96871e17');return p}
function approved(p,path,stage){const order=['taxonomy','study','cutover'],at=order.indexOf(stage);assert(at>=0,'unknown topic stage');return order.slice(0,at+1).map(s=>p.exceptions[path]?.[s]?.sha256).filter(Boolean)}
// 旧门禁仍使用旧基线；只有已登记准确目标摘要的新增改动可恢复至审核前字节再比较。
export function wrapTopicCompatibilityReader(reader,stage=process.env.TOPIC_COMPATIBILITY_STAGE??'taxonomy'){
 const p=policy(reader);return path=>{const bytes=reader(path),current=sha(bytes),snapshot=p.legacySnapshots[path];if(snapshot&&approved(p,path,stage).includes(current)&&current!==p.files[path]){const original=Buffer.from(snapshot.base64,'base64');assert.equal(sha(original),snapshot.sha256,path+' original snapshot');assert.equal(snapshot.sha256,p.files[path],path+' original baseline');return original}return bytes}
}
function inventory(root){const skip=new Set(['.git','.superpowers','.agents','.codex','node_modules','.next','bin','test-results','playwright-report','coverage']);const out=[];function walk(folder){for(const entry of readdirSync(folder,{withFileTypes:true})){if(skip.has(entry.name))continue;const path=join(folder,entry.name);if(entry.isDirectory())walk(path);else out.push(relative(root,path).replaceAll('\\','/'))}}walk(root);return out}
export function verifyTopicCompatibility({root=fileURLToPath(new URL('../../',import.meta.url)),stage=process.env.TOPIC_COMPATIBILITY_STAGE??'taxonomy'}={}){
 root=resolve(root);const reader=path=>{assert(!path.startsWith('/')&&!path.split('/').includes('..'),'unsafe baseline path');const full=join(root,path);assert(lstatSync(full).isFile(),path+' must be a regular file');return readFileSync(full)};const p=policy(reader),changed=[];
 for(const[path,original]of Object.entries(p.files)){const actual=sha(reader(path));assert(actual===original||approved(p,path,stage).includes(actual),'unapproved bytes: '+path);if(actual!==original)changed.push(path)}
 const api=JSON.parse(reader('api/openapi.yaml'));for(const[section,entries]of Object.entries(p.apiSections)){const values=section==='paths'?api.paths:api.components[section];for(const[name,original]of Object.entries(entries)){const target=p.apiExceptions[section+'/'+name]?.[stage];assert.equal(digest(values?.[name]),target??original,'original API: '+section+'/'+name)}}
 // 仅检查 Git 当前可见文件；构建产物和私有测试状态由现有 ignore 规则排除。
 const allowed=new Set(Object.keys(p.files));for(const s of ['taxonomy','study','cutover']){for(const path of p.newFiles[s]??[])allowed.add(path);if(s===stage)break}
 // 无Git的离线审查副本也执行同一文件名白名单。
 const names=gitInventory(root)??inventory(root);for(const path of names)assert(allowed.has(path),'unlisted topic file: '+path);
 const originalMigrations=Object.keys(p.files).filter(path=>/^db\/migrations\/0000[1-9]_.*\.sql$/.test(path)).length;assert.equal(originalMigrations,9);return {ok:true,stage,baseCommit:p.baseCommit,checkedFiles:Object.keys(p.files).length,originalMigrations,approvedChanges:changed}
}
import{spawnSync as run}from'node:child_process';
function gitInventory(root){const r=run('git',['-C',root,'ls-files','--cached','--others','--exclude-standard','-z'],{encoding:'utf8',maxBuffer:4<<20});return r.status===0?r.stdout.split('\0').filter(Boolean):null}
if(process.argv[1]&&resolve(process.argv[1])===fileURLToPath(import.meta.url)){const report=verifyTopicCompatibility({stage:process.argv[2]??'taxonomy'});process.stdout.write(JSON.stringify(report)+'\n')}

export function inverseTopicWorkflow(source,path){
 const root=fileURLToPath(new URL('../../',import.meta.url)),p=policy(name=>readFileSync(join(root,name)));const snapshot=p.legacySnapshots[path];if(snapshot&&['taxonomy','study','cutover'].some(stage=>approved(p,path,stage).includes(sha(Buffer.from(source))))){const old=Buffer.from(snapshot.base64,'base64');assert.equal(sha(old),p.files[path],path+' original workflow');return old.toString('utf8')}return source;
}
