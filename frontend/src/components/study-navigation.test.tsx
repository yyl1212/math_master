import {expect,it} from 'vitest';
import {render,screen,within} from '@testing-library/react';
import {UiLocaleProvider} from '@/lib/i18n/provider';
import {LearningHeader} from '@/features/study/managed-learning-view';
import {ReviewHeader} from '@/features/study-review/queue';

const hrefs=['/learn','/learn?view=notes','/learn/review','/learn/review/history','/learn?view=history'];
const headers=[<LearningHeader key="knowledge"/>,<LearningHeader key="notes" notes/>,<ReviewHeader key="review"/>,<ReviewHeader key="ratings" history/>,<LearningHeader key="history" history/>];
for(const locale of ['zh-CN','en'] as const){
 const labels=locale==='zh-CN'?['知识学习','我的笔记','复习工作区','自评历史','学习历史']:['Knowledge learning','My notes','Review workspace','Self-rating history','Learning history'];
 headers.forEach((header,index)=>it(`${locale}: ${hrefs[index]} keeps the same ordered workspace links and only marks its own section`,()=>{
  render(<UiLocaleProvider initialLocale={locale}>{header}</UiLocaleProvider>);
  const nav=screen.getByRole('navigation',{name:locale==='zh-CN'?'我的学习栏目':'My learning sections'});
  const links=within(nav).getAllByRole('link');
  expect(links.map(link=>link.getAttribute('href'))).toEqual(hrefs);
  expect(links.map(link=>link.textContent)).toEqual(labels);
  expect(links.filter(link=>link.getAttribute('aria-current')==='page')).toEqual([links[index]]);
 }));
}
