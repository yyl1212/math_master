import {z} from 'zod';
export const numberedFields=(max=50)=>({total:z.number().int().min(0).max(Number.MAX_SAFE_INTEGER),limit:z.number().int().min(1).max(max),offset:z.number().int().min(0).max(Number.MAX_SAFE_INTEGER)});
export type Numbered<T> = Omit<T,'nextCursor'> & {total:number;limit:number;offset:number};
