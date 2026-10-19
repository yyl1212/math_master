export type PageMeta={total:number;limit:number;offset:number};
export function pageModel({total,limit,offset}:PageMeta){
 if(![total,limit,offset].every(Number.isSafeInteger)||total<0||limit<1||offset<0)throw Error('Invalid page metadata');
 const totalPages=total?Math.floor((total-1)/limit)+1:0,lastOffset=totalPages?(totalPages-1)*limit:0;
 return {currentPage:total?Math.floor(Math.min(offset,lastOffset)/limit)+1:0,totalPages,lastOffset,needsCorrection:offset>lastOffset};
}
