import 'server-only';
import Link from 'next/link';
import {readManaged} from '@/lib/knowledge-admin/server-client';
import type {TopicPage,PublicPage} from '@/lib/knowledge-admin/types';
import {ContentState} from '@/components/content-state';
import {UiText} from '@/lib/i18n/ui-text';
import {uiMessage} from '@/lib/i18n/format';
import {LocalizedName} from '@/lib/i18n/localized-name';
import {UiPageTitle} from '@/components/ui-page-title';
import styles from '@/styles/current-home.module.css';
export async function CurrentHome({signedIn}:{signedIn:boolean}){
 const[topics,knowledge]=await Promise.all([readManaged<TopicPage>('/api/v3/topics?limit=100'),readManaged<PublicPage>('/api/v3/knowledge?limit=1')]);
 if(!topics.ok||!knowledge.ok)return <ContentState kind="unavailable"/>;
 const published=topics.data.items.filter(item=>item.knowledgeCount>0),overview=(published.length?published:topics.data.items).slice(0,6);
 return <div className={styles.page}><UiPageTitle messageKey="page.home"/>
  <header className={styles.hero}><p className="eyebrow">Math Master</p><h1><UiText notice={uiMessage('streamline.home.title',{})}/></h1><p><UiText notice={uiMessage('streamline.home.intro',{})}/></p><div className={styles.actions}><Link className="button" prefetch={false} href={signedIn?'/learn':'/login'}><UiText notice={uiMessage(signedIn?'streamline.home.continue':'streamline.home.start',{})}/></Link><Link prefetch={false} href="/knowledge"><UiText notice={uiMessage('streamline.home.map',{})}/> →</Link></div></header>
  <section className={styles.summary}><p><UiText notice={uiMessage('managed.taxonomy',{})}/></p><p><UiText notice={uiMessage('streamline.home.available',{count:knowledge.data.total})}/></p></section>
  <section><div className={styles.heading}><h2><UiText notice={uiMessage('streamline.home.themes',{})}/></h2><Link prefetch={false} href="/knowledge"><UiText notice={uiMessage('streamline.home.allThemes',{})}/> →</Link></div><div className="domain-grid">{overview.map(topic=><article className="domain-card" key={topic.topicKey}><h3><Link prefetch={false} href={'/topics/'+(topic.topicKey==='project:other'?'project-other':topic.topicKey)}><LocalizedName english={topic.titleEn} chinese={topic.title}/></Link></h3><p className={styles.cardMeta}><UiText notice={uiMessage('streamline.home.available',{count:topic.knowledgeCount})}/></p></article>)}</div></section>
  <section className={styles.steps}><h2><UiText notice={uiMessage('streamline.home.how',{})}/></h2><ol><li><UiText notice={uiMessage('streamline.home.step1',{})}/></li><li><UiText notice={uiMessage('streamline.home.step2',{})}/></li><li><UiText notice={uiMessage('streamline.home.step3',{})}/></li></ol></section>
 </div>;
}
