import {expect,it} from 'vitest';
import {pageModel} from './model';
it('derives empty, single, middle and corrected last pages',()=>{
 expect(pageModel({total:0,limit:20,offset:0})).toMatchObject({currentPage:0,totalPages:0});
 expect(pageModel({total:1,limit:20,offset:0})).toMatchObject({currentPage:1,totalPages:1});
 expect(pageModel({total:41,limit:20,offset:20})).toMatchObject({currentPage:2,totalPages:3,lastOffset:40,needsCorrection:false});
 expect(pageModel({total:41,limit:20,offset:100})).toMatchObject({currentPage:3,lastOffset:40,needsCorrection:true});
 expect(()=>pageModel({total:1,limit:0,offset:0})).toThrow();
});
