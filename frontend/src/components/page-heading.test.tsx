import {it,expect} from 'vitest';
import {render,screen} from '@testing-library/react';
import {PageHeading} from './page-heading';
import {LocalizedName} from '@/lib/i18n/localized-name';
it('保留一个一级标题和原链接名称，空槽不占布局',()=>{
 const {container,rerender}=render(<PageHeading title={<LocalizedName english="Original mathematics" chinese="原创数学"/>}/>);
 expect(screen.getAllByRole('heading',{level:1})).toHaveLength(1);expect(screen.getByRole('heading',{level:1})).toHaveTextContent('Original mathematics');
 expect(container.querySelector('header')?.children).toHaveLength(1);
 rerender(<PageHeading title="Notes" description="Original description" meta={0} actions={<a href="/learn">My learning</a>}/>);
 expect(screen.getAllByRole('heading',{level:1})).toHaveLength(1);expect(screen.getByText('Original description')).toBeVisible();expect(screen.getByText('0')).toBeVisible();expect(screen.getByRole('link',{name:'My learning'})).toHaveAttribute('href','/learn');
 rerender(<PageHeading title="Notes" description="" meta={false} actions={null}/>);expect(container.querySelector('header')?.children).toHaveLength(1);
});
