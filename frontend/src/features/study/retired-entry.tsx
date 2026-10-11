import Link from 'next/link';
import {UiText} from '@/lib/i18n/ui-text';
import {uiMessage} from '@/lib/i18n/format';
export function RetiredEntry(){return <section className="content-state"><h1><UiText notice={uiMessage('streamline.retired.title',{})}/></h1><p><UiText notice={uiMessage('streamline.retired.body',{})}/></p><div className="actions"><Link prefetch={false} href="/learn"><UiText notice={uiMessage('managed.learn',{})}/></Link><Link prefetch={false} href="/knowledge"><UiText notice={uiMessage('managed.map',{})}/></Link></div></section>}
