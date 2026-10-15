import {expect,it} from 'vitest';
import {fireEvent,render,screen,within} from '@testing-library/react';
import {UiLocaleProvider,useUiI18n} from '@/lib/i18n/provider';
import {StudyNavigation} from './study-navigation';

function LanguageSwitch(){const{setLocale}=useUiI18n();return <button onClick={()=>setLocale('en')}>English</button>;}
it('keeps link order, current section and the same link nodes across a language change',()=>{
 render(<UiLocaleProvider initialLocale="zh-CN"><LanguageSwitch/><StudyNavigation active="notes"/></UiLocaleProvider>);
 const hrefs=['/learn','/learn?view=notes','/learn/review','/learn/review/history','/learn?view=history'];
 const nav=screen.getByRole('navigation',{name:'我的学习栏目'}),links=within(nav).getAllByRole('link');
 expect(links.map(link=>link.getAttribute('href'))).toEqual(hrefs);
 expect(links.map(link=>link.textContent)).toEqual(['知识学习','我的笔记','复习工作区','自评历史','学习历史']);
 expect(links.filter(link=>link.getAttribute('aria-current')==='page')).toEqual([links[1]]);
 fireEvent.click(screen.getByRole('button',{name:'English'}));
 const translated=within(screen.getByRole('navigation',{name:'My learning sections'})).getAllByRole('link');
 translated.forEach((link,index)=>expect(link).toBe(links[index]));
 expect(translated.map(link=>link.getAttribute('href'))).toEqual(hrefs);
 expect(translated.map(link=>link.textContent)).toEqual(['Knowledge learning','My notes','Review workspace','Self-rating history','Learning history']);
 expect(translated.filter(link=>link.getAttribute('aria-current')==='page')).toEqual([links[1]]);
});
