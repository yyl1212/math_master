import{spawnSync}from'node:child_process';import{resolve}from'node:path';import{fileURLToPath}from'node:url';
export function partitionTests(names,{size=70,skip=''}={}){
 if(!Number.isInteger(size)||size<1||size>80||names.some(n=>!/^(Test|Fuzz|Example)[A-Za-z0-9_]*$/.test(n))||new Set(names).size!==names.length)throw Error('Invalid Go test partition');
 const excluded=new RegExp(skip||'(?!)'),selected=names.filter(n=>!excluded.test(n)).sort(),batches=[];for(let i=0;i<selected.length;i+=size)batches.push(selected.slice(i,i+size));return{selected,batches};
}
function main(args){
 const opts={batch:0,size:70,slots:4,skip:'',plan:false},seen=new Set();
 for(let i=0;i<args.length;i++){const key=args[i].replace(/^--/,'');if(!args[i].startsWith('--')||!Object.hasOwn(opts,key)||seen.has(key))throw Error('Invalid batch option');seen.add(key);if(key==='plan'){opts.plan=true;continue}const value=args[++i];if(value===undefined)throw Error('Missing batch option');opts[key]=key==='skip'?value:Number(value)}
 if(!Number.isInteger(opts.batch)||!Number.isInteger(opts.slots)||opts.batch<0||opts.batch>=opts.slots||opts.slots<1||opts.slots>8)throw Error('Invalid batch bounds');
 const listing=spawnSync('go',['test','./internal/store','-list','.','-timeout','5m'],{encoding:'utf8',maxBuffer:2<<20,timeout:300000});if(listing.status!==0)throw Error('Go test inventory failed');
 const result=partitionTests(listing.stdout.split(/\r?\n/).filter(n=>/^(Test|Fuzz|Example)/.test(n)),{size:opts.size,skip:opts.skip});if(result.batches.length>opts.slots)throw Error('Configured batches omit tests');
 if(opts.plan){process.stdout.write(JSON.stringify({selected:result.selected.length,batches:result.batches.map(b=>b.length),slots:opts.slots})+'\n');return}
 const batch=result.batches[opts.batch]??[];process.stdout.write(`Go batch ${opts.batch+1}/${opts.slots}: ${batch.length} of ${result.selected.length} selected tests\n`);if(!batch.length)return;
 const r=spawnSync('go',['test','./internal/store','-run','^('+batch.join('|')+')$','-timeout','5m','-count=1'],{stdio:'inherit',timeout:310000});process.exitCode=r.status??1;
}
if(process.argv[1]&&resolve(process.argv[1])===fileURLToPath(import.meta.url)){try{main(process.argv.slice(2))}catch{console.error('Go batch validation failed');process.exitCode=1}}
