export function pageHref(pathname:string,query:Record<string,string>,offset:number,options:{offsetKey:string;pageMode:boolean;limit:number;pageKey?:string;cursorKey?:string}){
 if(!pathname.startsWith('/')||pathname.startsWith('//')||!Number.isSafeInteger(offset)||offset<0||!Number.isSafeInteger(options.limit)||options.limit<1)throw Error('Invalid page navigation');
 const q=new URLSearchParams(query);
 if(options.pageMode){q.delete(options.offsetKey);q.delete(options.cursorKey??'cursor');q.set(options.pageKey??'page',String(Math.floor(offset/options.limit)+1));}else q.set(options.offsetKey,String(offset));
 return pathname+'?'+q;
}
export type NumberedQuery={page?:number;limit?:number;cursor?:string};
export function numberedPath(pathname:string,query:NumberedQuery={}){
 if(Object.keys(query).some(k=>!['page','limit','cursor'].includes(k))||query.page!==undefined&&(!Number.isSafeInteger(query.page)||query.page<1||query.cursor!==undefined)||query.limit!==undefined&&(!Number.isSafeInteger(query.limit)||query.limit<1||query.limit>50))throw Error('Invalid numbered query');
 const params=new URLSearchParams({pagination:'page'});for(const[k,v]of Object.entries(query))if(v!==undefined)params.set(k,String(v));return pathname+'?'+params;
}
