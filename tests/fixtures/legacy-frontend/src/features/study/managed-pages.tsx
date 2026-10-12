import 'server-only';import {headers} from 'next/headers';import {readServerSession} from '@/lib/auth/server-client';import {readManaged} from '@/lib/knowledge-admin/server-client';import {readServerStudy} from '@/lib/study/server-client';import type {HistoryPage as LegacyHistoryPage} from '@/lib/study/types';import type {StudyPage,Overview,ProgressPage,HistoryPage,StudyDetail,Note,TopicPage,Topic} from '@/lib/knowledge-admin/types';import {ContentState} from '@/components/content-state';import {StudyAccountBoundary} from './account-boundary';import {ManagedControls} from './managed-controls';import {ManagedHistory} from './managed-history';import {ManagedLearningView,type LearningTheme} from './managed-learning-view';

async function actor(){const cookie=(await headers()).get('cookie')??'',user=await readServerSession(cookie);return user.ok&&user.data&&!user.data.mustChangePassword?{cookie,user:user.data}:null}
function learningTopicKey(value:string){
 const key=value.replace(/^msc-/i,'').toUpperCase().replace(/-?XX$/,'');
 if(key==='PROJECT:OTHER'||key==='PROJECT-OTHER')return 'project:other';
 if(/^\d{2}$/.test(key))return key+'-XX';
 if(/^\d{2}[A-Z]$/.test(key))return key+'xx';
 return /^\d{2}[A-Z]\d{2}$/.test(key)?key:value;
}
export async function ManagedLearnPage({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){
 const a=await actor();if(!a)return <ContentState kind="unavailable"/>;
 const p=await searchParams,filters:Record<string,string>={},q=new URLSearchParams();
 for(const k of ['q','topicKey','state','offset'])if(typeof p[k]==='string'&&p[k]){filters[k]=k==='topicKey'?learningTopicKey(p[k]):p[k];q.set(k,filters[k]);}
 if(p.mode==='review'){filters.mode='review';q.set('reviewOnly','true');}
 const[o,list,progress,directory]=await Promise.all([
  readManaged<Overview>('/api/v3/study/overview',a.cookie),
  readManaged<StudyPage>('/api/v3/study/knowledge'+(q.size?'?'+q:''),a.cookie),
  readManaged<ProgressPage>('/api/v3/study/topics?limit=100',a.cookie),
  readManaged<TopicPage>('/api/v3/topics?limit=100')
 ]);
 if(!o.ok||!list.ok||!progress.ok||!directory.ok||o.data.actorId!==a.user.id||list.data.items.some(d=>d.actorId!==a.user.id))return <ContentState kind="unavailable"/>;
 const names=new Map(directory.data.items.map(topic=>[topic.topicKey,topic]));
 const active=progress.data.items.filter(topic=>topic.total>0);
 if(active.some(topic=>!names.has(topic.topicKey)))return <ContentState kind="unavailable"/>;
 const themes:LearningTheme[]=active.map(topic=>({...topic,title:names.get(topic.topicKey)!.title,titleEn:names.get(topic.topicKey)!.titleEn}));
 let selectedTopic:Topic|undefined=names.get(filters.topicKey);
 if(filters.topicKey&&!names.has(filters.topicKey)){
  const selected=await readManaged<Topic>('/api/v3/topics/'+encodeURIComponent(filters.topicKey));
  if(!selected.ok)return <ContentState kind="unavailable"/>;selectedTopic=selected.data;
 }
 return <StudyAccountBoundary actorId={a.user.id}><ManagedLearningView overview={o.data} list={list.data} themes={themes} filters={filters} selectedTopic={selectedTopic}/></StudyAccountBoundary>;
}
export async function ManagedHistoryPage({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){const a=await actor();if(!a)return <ContentState kind="unavailable"/>;const p=await searchParams,q=new URLSearchParams();for(const k of ['knowledgeId','topicKey','kind','from','to','cursor']){if(typeof p[k]==='string'&&p[k])q.set(k,p[k])};const [r,legacy]=await Promise.all([readManaged<HistoryPage>('/api/v3/study/history'+(q.size?'?'+q:''),a.cookie),readServerStudy<LegacyHistoryPage>({kind:'history',query:{...(typeof p.legacyCursor==='string'?{cursor:p.legacyCursor}:{}),limit:50}},a.cookie)]);return r.ok&&r.data.actorId===a.user.id?<StudyAccountBoundary actorId={a.user.id}><ManagedHistory page={r.data} filters={Object.fromEntries(q)} legacy={legacy.ok&&legacy.data.actorId===a.user.id?legacy.data:undefined}/></StudyAccountBoundary>:<ContentState kind="unavailable"/>}
export async function ManagedPersonal({id}:{id:string}){const a=await actor();if(!a)return null;const[d,n]=await Promise.all([readManaged<StudyDetail>(`/api/v3/study/knowledge/${id}`,a.cookie),readManaged<Note>(`/api/v3/study/knowledge/${id}/note`,a.cookie)]);return d.ok&&d.data.actorId===a.user.id?<StudyAccountBoundary actorId={a.user.id}><ManagedControls initial={d.data} note={n.ok&&n.data.actorId===a.user.id?n.data:undefined}/></StudyAccountBoundary>:null}
