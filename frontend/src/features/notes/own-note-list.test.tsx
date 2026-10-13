import {it,expect} from 'vitest';import {render,screen} from '@testing-library/react';import {OwnNoteList} from './own-note-list';import type {OwnNotePage} from '@/lib/note-sharing/types';
const actor='51390b7d-a8e9-4e98-a3ae-22101c20b309',kid='k-'+'a'.repeat(56);
it('shows an unavailable note as retained read-only and keeps filters in paging',()=>{
 const page:OwnNotePage={actorId:actor,items:[{knowledgeId:kid,knowledgeTitle:'Fractions',knowledgeTitleZh:'分数',topicKeys:['97F40'],excerpt:'Own retained note',updatedAt:'2026-10-13T00:00:00Z',available:false,noteRevision:1,share:{actorId:actor,knowledgeId:kid,status:'withdrawn',sequence:2,submissionId:actor,approvedAt:null,lastDecisionReason:''}}],nextCursor:'cursor'};
 render(<OwnNoteList page={page} filters={{q:'fractions',shareStatus:'withdrawn'}}/>);expect(screen.getByText('Own retained note')).toBeVisible();expect(screen.getByText('Knowledge is unavailable; your note is retained.')).toBeVisible();expect(screen.getByRole('link',{name:'Next page'})).toHaveAttribute('href',expect.stringContaining('q=fractions'));expect(screen.getByRole('link',{name:'View note'})).toHaveAttribute('href','/knowledge/'+kid+'?mode=notes');
});
