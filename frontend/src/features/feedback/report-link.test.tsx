import {it,expect} from 'vitest';
import {render,screen} from '@testing-library/react';
import {ReportLink} from './report-link';
it('locates real source IDs and exact assessment position without a client identity',()=>{render(<ReportLink source={{kind:'assessment',id:'11111111-1111-4111-8111-111111111111',position:5,partKind:'asset',partId:'halves'}}/>);const link=screen.getByRole('link',{name:'Report a problem'});expect(link.getAttribute('href')).toBe('/feedback/new?kind=assessment&id=11111111-1111-4111-8111-111111111111&position=5&partKind=asset&partId=halves');expect(link.getAttribute('href')).not.toContain('actor')});
it('site uses a fixed area without a made-up mathematical version',()=>{render(<ReportLink source={{kind:'site',area:'home'}}/>);expect(screen.getByRole('link').getAttribute('href')).toBe('/feedback/new?kind=site&area=home')});

it('topic feedback keeps taxonomy identity as website location rather than mathematical authority',()=>{render(<ReportLink source={{kind:'site',area:'other'}} topic={{topicId:'msc-00a00',taxonomyVersionId:'a'.repeat(64)}}/>);expect(screen.getByRole('link').getAttribute('href')).toBe('/feedback/new?kind=site&area=other&topicId=msc-00a00&taxonomyVersionId='+'a'.repeat(64))});
