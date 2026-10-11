import 'server-only';
import Link from 'next/link';
import {readManaged} from '@/lib/knowledge-admin/server-client';
import type {TopicPage,Topic,PublicPage} from '@/lib/knowledge-admin/types';
import {CatalogueSearch} from './catalogue-search';
import {ContentState} from '@/components/content-state';
import {UiText} from '@/lib/i18n/ui-text';
import {LocalizedName} from '@/lib/i18n/localized-name';
import {uiMessage} from '@/lib/i18n/format';
const url=(key:string)=>`/topics/${key==='project:other'?'project-other':key}`;
export const pageOffset=(value:unknown)=>typeof value==='string'&&/^\d{1,6}$/.test(value)?Number(value):0;
function CatalogueAttribution(){
 return <p className="map-note"><UiText notice={uiMessage('topic.map.translationNote',{})}/>{' '}
  <a href="https://msc2020.org/" rel="noreferrer">MSC2020</a>{' · '}
  <a href="https://minsecrus.github.io/learning-notes/notes/2026/07/23/msc2020-mathematics-subject-classification-zh" rel="noreferrer">Learning Notes / minsecrus</a>{' · '}
  <a href="https://creativecommons.org/licenses/by-nc-sa/4.0/" rel="noreferrer">CC BY-NC-SA 4.0</a>
 </p>;
}
function PublishedCount({count}:{count:number}){
 return <UiText notice={uiMessage(count===1?'public.publishedOne':'public.publishedMany',{count})}/>;
}
function TopicCard({topic}:{topic:Topic}){
 return <Link className="domain-card" prefetch={false} href={url(topic.topicKey)}><h3><LocalizedName english={topic.titleEn} chinese={topic.title}/></h3><p><PublishedCount count={topic.knowledgeCount}/></p></Link>;
}
function Paging({page,path,query,keyName='offset'}:{page:{total:number;offset:number;limit:number};path:string;query:URLSearchParams;keyName?:string}){
 const link=(offset:number)=>{const next=new URLSearchParams(query);next.set(keyName,String(offset));return path+'?'+next};
 return <nav>{page.offset>0&&<Link prefetch={false} href={link(Math.max(0,page.offset-page.limit))}><UiText notice={uiMessage('managed.previous',{})}/></Link>} {' '}{page.offset+page.limit<page.total&&<Link prefetch={false} href={link(page.offset+page.limit)}><UiText notice={uiMessage('managed.next',{})}/></Link>}</nav>;
}
function KnowledgeCards({items}:{items:PublicPage['items']}){
 return <div className="domain-grid">{items.map(knowledge=><article className="domain-card" key={knowledge.id}><h3><Link href={`/knowledge/${knowledge.id}`} prefetch={false}><LocalizedName english={knowledge.point.title} chinese={knowledge.point.title_zh}/></Link></h3><p><UiText notice={uiMessage(`managed.type.${knowledge.point.type}`,{})}/> · <UiText notice={uiMessage('managed.difficulty',{})}/>: {knowledge.point.learning_difficulty.difficulty_level}</p></article>)}</div>;
}
export async function CurrentCatalogue({topicKey,q,offset=0,mode='topics'}:{topicKey?:string;q?:string;offset?:number;mode?:'topics'|'knowledge'}){
 const keyword=(q??'').trim(),scope=topicKey?.trim()||undefined;
 const start=q!==undefined&&!keyword?0:offset;
 const query=new URLSearchParams({limit:mode==='knowledge'?'20':'100',offset:String(start)});if(scope)query.set('topicKey',scope);if(keyword)query.set('q',keyword);
 const [topics,knowledge,scopeInfo]=await Promise.all([
  mode==='topics'?readManaged<TopicPage>('/api/v3/topics?'+query):Promise.resolve(null),
  mode==='knowledge'?readManaged<PublicPage>('/api/v3/knowledge?'+query):Promise.resolve(null),
  mode==='knowledge'&&scope?readManaged<Topic>('/api/v3/topics/'+encodeURIComponent(scope)):Promise.resolve(null)
 ]);
 const result=topics??knowledge!;
 const paging=new URLSearchParams({mode});if(scope)paging.set('topicKey',scope);if(keyword)paging.set('q',keyword);
 const kind=topics?.ok?topics.data.items.find(topic=>topic.kind!=='project-other')?.kind:undefined;
 const level=kind==='secondary'?'topic.map.level2':kind==='specific'||kind==='other'?'topic.map.level3':'topic.map.level1';
 return <><header className="page-heading"><h1><UiText notice={uiMessage('managed.map',{})}/></h1><p><UiText notice={uiMessage('managed.taxonomy',{})}/></p></header>
  <CatalogueSearch key={mode+'|'+(scope??'')+'|'+keyword} initialMode={mode} initialQuery={keyword} topicKey={scopeInfo?.ok?scopeInfo.data.topicKey:scope}/>
  {scopeInfo?.ok&&<p><UiText notice={uiMessage('map.search.scope',{})}/>: <LocalizedName english={scopeInfo.data.titleEn} chinese={scopeInfo.data.title}/></p>}
  {!result.ok?(result.status===422?<p role="alert"><UiText notice={uiMessage('managed.invalidInput',{})}/></p>:<ContentState kind="unavailable"/>):<>
   {topics?.ok&&<section><h2><UiText notice={uiMessage(level,{})}/></h2><div className="domain-grid">{topics.data.items.map(topic=><TopicCard key={topic.topicKey} topic={topic}/>)}</div>{topics.data.items.length===0&&<p role="status"><UiText notice={uiMessage('map.search.noTopics',{})}/></p>}</section>}
   {knowledge?.ok&&<section><h2><UiText notice={uiMessage('topic.map.knowledge',{})}/></h2><KnowledgeCards items={knowledge.data.items}/>{knowledge.data.items.length===0&&<p role="status"><UiText notice={uiMessage(keyword?'topic.map.noKnowledgeResults':'map.search.noKnowledge',{})}/></p>}</section>}
   <Paging page={result.data} path="/knowledge" query={paging}/>
  </>}<CatalogueAttribution/></>;
}
export async function CurrentTopicPage({topicKey,searchParams={}}:{topicKey:string;searchParams?:Record<string,string|string[]|undefined>}){
 const keyword=typeof searchParams.q==='string'?searchParams.q.trim():'';
 const q=new URLSearchParams({limit:'20',offset:String(typeof searchParams.q==='string'&&!keyword?0:pageOffset(searchParams.knowledgeOffset))});if(keyword)q.set('q',keyword);
 let topic=await readManaged<Topic>('/api/v3/topics/'+encodeURIComponent(topicKey)+'?'+q);
 const invalidInput=!topic.ok&&topic.status===422;
 if(invalidInput)topic=await readManaged<Topic>('/api/v3/topics/'+encodeURIComponent(topicKey)+'?limit=20&offset=0');
 if(!topic.ok)return <ContentState kind={topic.status===404?'not-found':'unavailable'}/>;
 const parent=topic.data,childQuery=new URLSearchParams({topicKey:parent.topicKey,limit:'100',offset:String(pageOffset(searchParams.offset))});
 const children=['primary','secondary'].includes(parent.kind)?await readManaged<TopicPage>('/api/v3/topics?'+childQuery):null;
 const query=new URLSearchParams();for(const [key,value]of Object.entries(searchParams))if(typeof value==='string'&&value.trim()&&!(invalidInput&&(key==='q'||key==='knowledgeOffset')))query.set(key,key==='q'?keyword:value);
 const searchLink=new URLSearchParams({mode:'knowledge',topicKey:parent.topicKey});if(keyword)searchLink.set('q',keyword);
 return <><header className="page-heading"><Link href="/knowledge" prefetch={false}><UiText notice={uiMessage('managed.map',{})}/></Link><h1><LocalizedName english={parent.titleEn} chinese={parent.title}/></h1><p><PublishedCount count={parent.knowledgeCount}/></p></header>
  {children&&!children.ok?<ContentState kind="unavailable"/>:children?.ok&&<section><h2><UiText notice={uiMessage(parent.kind==='primary'?'topic.map.level2':'topic.map.level3',{})}/></h2>
   <div className="domain-grid">{children.data.items.map(child=><TopicCard key={child.topicKey} topic={child}/>)}</div>
   <Paging page={children.data} path={url(topicKey)} query={query}/></section>}
  <section><h2><UiText notice={uiMessage('topic.map.knowledge',{})}/></h2>
   {invalidInput&&<p role="alert"><UiText notice={uiMessage('managed.invalidInput',{})}/></p>}
   <p><Link className="button secondary" href={'/knowledge?'+searchLink} prefetch={false}><UiText notice={uiMessage('map.search.within',{})}/></Link></p>
   <KnowledgeCards items={parent.items.items}/>
   {parent.items.total===0&&<p><UiText notice={uiMessage(keyword&&!invalidInput?'topic.map.noKnowledgeResults':'topic.map.empty',{})}/></p>}
   <Paging page={parent.items} path={url(topicKey)} query={query} keyName="knowledgeOffset"/></section><CatalogueAttribution/></>;
}
