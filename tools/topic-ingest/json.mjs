import {TextDecoder} from 'node:util';
const decoder = new TextDecoder('utf-8',{fatal:true});
const invalid = () => { throw new Error('INVALID_SOURCE_JSON'); };
function validString(value) {
  if (value.includes('\0')) invalid();
  for (let i=0;i<value.length;i++) {
    const c=value.charCodeAt(i);
    if(c>=0xd800&&c<=0xdbff){const n=value.charCodeAt(++i);if(!(n>=0xdc00&&n<=0xdfff))invalid();}
    else if(c>=0xdc00&&c<=0xdfff)invalid();
  }
}
export function parseSourceJSON(bytes, options = {}) {
  let source;try{source=decoder.decode(bytes);}catch{invalid();}
  let i=0;
  const space=()=>{while(/[ \t\r\n]/.test(source[i]??'!'))i++;};
  function string() {
    const start=i++;let escaped=false;
    while(i<source.length){
      const c=source[i++];
      if(!escaped&&c==='"'){let v;try{v=JSON.parse(source.slice(start,i));}catch{invalid();}validString(v);return v;}
      if(!escaped&&c==='\\')escaped=true;else escaped=false;
    }
    invalid();
  }
  function value(depth) {
    if(depth>(options.exact ? 32 : 128))invalid();space();const c=source[i];
    if(c==='"')return string();
    if(c==='{'){
      i++;const keys=new Set();space();if(source[i]==='}'){i++;return;}
      while(true){
        space();if(source[i]!=='"')invalid();const key=string();
        const identity=options.exact ? key : key.toLowerCase();if(keys.has(identity)||!options.exact && ['__proto__','constructor','prototype'].includes(identity))invalid();keys.add(identity);
        space();if(source[i++]!==':')invalid();value(depth+1);space();
        const next=source[i++];if(next==='}')return;if(next!==',')invalid();
      }
    }
    if(c==='['){i++;space();if(source[i]===']'){i++;return;}while(true){value(depth+1);space();const next=source[i++];if(next===']')return;if(next!==',')invalid();}}
    const match=/^(?:true|false|null|-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?)/.exec(source.slice(i));
    if(!match)invalid();if(/^[-\d]/.test(match[0])&&(!Number.isFinite(Number(match[0]))||options.exact&&!/[.eE]/.test(match[0])&&!Number.isSafeInteger(Number(match[0]))))invalid();i+=match[0].length;
  }
  value(0);space();if(i!==source.length)invalid();
  try{return JSON.parse(source);}catch{invalid();}
}

export function parseManagedJSON(bytes) { return parseSourceJSON(bytes, {exact:true}); }
