export function pageHref(pathname:string,query:Record<string,string>,offset:number,options:{offsetKey:string;pageMode:boolean;limit:number;pageKey?:string;cursorKey?:string}){
 if(!pathname.startsWith('/')||pathname.startsWith('//')||!Number.isSafeInteger(offset)||offset<0||!Number.isSafeInteger(options.limit)||options.limit<1)throw Error('Invalid page navigation');
 const q=new URLSearchParams(query);
 if(options.pageMode){q.delete(options.cursorKey??'cursor');q.set(options.pageKey??'page',String(Math.floor(offset/options.limit)+1));}else q.set(options.offsetKey,String(offset));
 return pathname+'?'+q;
}
