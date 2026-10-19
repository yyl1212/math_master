import {expect,it} from 'vitest';
import {pageHref} from './query';
it('changes only the addressed paging window',()=>{
 expect(pageHref('/topics/97',{offset:'100',q:'sum',knowledgeOffset:'0'},40,{offsetKey:'knowledgeOffset',pageMode:false,limit:20})).toBe('/topics/97?offset=100&q=sum&knowledgeOffset=40');
 expect(pageHref('/learn',{view:'history',page:'2',legacyCursor:'old'},40,{offsetKey:'legacyOffset',pageMode:true,limit:20,pageKey:'legacyPage',cursorKey:'legacyCursor'})).toBe('/learn?view=history&page=2&legacyPage=3');
});
