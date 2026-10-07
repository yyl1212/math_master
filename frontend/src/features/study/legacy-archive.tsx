import Link from "next/link";
import {UiText} from "@/lib/i18n/ui-text";
import {uiMessage} from "@/lib/i18n/format";
import {learningActor} from "@/features/learning/page-data";
import {LearningBoundary,LearningNotice} from "@/features/learning/learning-status";
import {HistoryList} from "@/features/learning/history-list";
import {ResultPanel,ResultItemPanel} from "@/features/assessment/result-panel";
import {getLearningClient} from "@/lib/learning/server-client";
import {learningFailure,learningUUID} from "@/lib/learning/schemas";
import {learningAwait} from "@/lib/learning/bytes";
export async function LegacyStudyArchive({searchParams}:{searchParams:Promise<Record<string,string|string[]|undefined>>}){
 const controller=new AbortController(),timer=setTimeout(()=>controller.abort(),10000);
 try{
  const q=await learningAwait(searchParams,controller.signal);if(q.archive!=="legacy"||Object.entries(q).some(([k,v])=>typeof v!=="string"||!["archive","attempt","module","offset","knowledge","version"].includes(k))||q.attempt!==undefined&&!learningUUID.test(String(q.attempt))||q.module!==undefined&&!['assessment','practice'].includes(String(q.module))||q.module!==undefined&&q.attempt===undefined||q.attempt!==undefined&&q.knowledge!==undefined||q.knowledge!==undefined&&!/^[a-z][a-z0-9-]{0,63}$/.test(String(q.knowledge))||(q.knowledge===undefined)!==(q.version===undefined)||q.version!==undefined&&!/^[1-9][0-9]{0,8}$/.test(String(q.version))||q.offset!==undefined&&!/^(?:0|[1-9][0-9]{0,4})$/.test(String(q.offset)))return <LearningNotice result={learningFailure("INVALID_REQUEST")}/>;
  const actor=await learningAwait(learningActor(),controller.signal);if(!actor.ok)return <LearningNotice result={actor.error}/>;const client=getLearningClient(controller.signal);let detail:React.ReactNode=null;
  if(q.attempt){const id=String(q.attempt);if(q.module==="practice"){const result=await client.readPractice(id);if(!result.ok)return <LearningNotice result={result}/>;detail=result.data.result?<ResultItemPanel item={result.data.result.item} attemptId={id} sourceKind="practice"/>:<p><UiText notice={uiMessage("study.archive.active",{})}/></p>}else{const attempt=await client.readAssessment(id);if(!attempt.ok)return <LearningNotice result={attempt}/>;if(attempt.data.summary.state==="submitted"){const result=await client.readAssessmentResult(id);if(!result.ok)return <LearningNotice result={result}/>;detail=<ResultPanel result={result.data}/>}else detail=<p><UiText notice={uiMessage("study.archive.active",{})}/></p>}}
  if(q.knowledge){const result=await client.readLearningKnowledge(String(q.knowledge),Number(q.version));if(!result.ok)return <LearningNotice result={result}/>;detail=<section className="panel"><h2>{result.data.state.title}</h2><p>v{result.data.state.knowledge.version}</p></section>}
  const history=await client.listLearningHistory({limit:20,offset:Number(q.offset??0)});if(!history.ok)return <LearningNotice result={history}/>;
  return <><header className="page-heading"><h1><UiText notice={uiMessage("study.archive.title",{})}/></h1><p><UiText notice={uiMessage("study.retired.body",{})}/></p><Link prefetch={false} href="/learn"><UiText notice={uiMessage("study.mode.learn",{})}/></Link></header><LearningBoundary actorId={actor.id}>{detail}<HistoryList page={history.data} archived/></LearningBoundary></>;
 }catch{return <LearningNotice result={learningFailure()}/>}finally{clearTimeout(timer)}
}
