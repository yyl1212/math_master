import {inverseTopicWorkflow} from "./topic-learning-compatibility.mjs";
export const languageBatches=['public','auth','content','question','learning','feedback','correction'].map(name=>['ui-language-'+name+'.spec.ts']);
export const originalBatches=[['catalogue','reading'],['auth','auth-security'],['content-authoring'],['content-review'],['content-release'],['content-security'],['question-authoring'],['question-review'],['question-release'],['question-security'],['learning-progress'],['learning-practice','learning-assessment'],['learning-diagnostic','learning-security'],['learning-review-regressions'],['feedback-user'],['feedback-review'],['feedback-security'],['correction-user'],['correction-review'],['notification-security'],['content-acceptance-reading'],['content-acceptance-learning']].map(batch=>batch.map(name=>name+'.spec.ts'));
export const e2eCommand=batch=>'node tools/verify/run.mjs --cwd frontend -- env -u NO_COLOR npm run e2e -- '+batch.join(' ');
/** Read only active YAML run scalars, excluding comments and step names. */
export function activeRuns(source){const lines=source.split('\n'),runs=[];for(let i=0;i<lines.length;i++){const m=lines[i].match(/^(\s*)run:\s*(.*)$/);if(!m)continue;if(m[2]!=='|'){runs.push(m[2]);continue;}const body=[];while(i+1<lines.length&&(!lines[i+1].trim()||lines[i+1].match(/^\s*/)[0].length>m[1].length))body.push(lines[++i].trim());runs.push(body.join('\n'));}return runs;}
export function validateUiLanguageCI(source,policy){
 if(policy.workers!==1||policy.retries!==0||policy.globalTimeout!==480000||JSON.stringify(policy.viewports)!==JSON.stringify([{width:1280,height:900},{width:390,height:844}]))throw Error('browser budget changed');
 const runs=activeRuns(source);
 for(const batch of [...originalBatches,...languageBatches])if(!runs.includes(e2eCommand(batch)))throw Error('missing isolated browser batch: '+batch.join(' '));
 return true;
}
/** This configuration contract intentionally permits only explicit literal budgets. */
export function readPlaywrightPolicy(source){
 // Budget declarations live on their own lines in the checked-in config.
 const number=name=>{const all=[...source.matchAll(new RegExp('^\\s*'+name+':\\s*([0-9]+)\\s*,?\\s*$','gm'))];if(all.length!==1)throw Error('ambiguous or nonliteral browser budget: '+name);return Number(all[0][1]);};
 const views=[...source.matchAll(/viewport:\s*\{\s*width:\s*(\d+),\s*height:\s*(\d+)\s*\}/g)].map(m=>({width:Number(m[1]),height:Number(m[2])}));
 return {workers:number('workers'),retries:number('retries'),globalTimeout:number('globalTimeout'),viewports:views};
}
const languageStepLabels=['公共阅读','账户','知识后台','题库','学习测评','反馈','纠错通知'];
export const languageWorkflowSteps=languageBatches.map((batch,i)=>'      - name: 中英文'+languageStepLabels[i]+'双视口验证\n        run: '+e2eCommand(batch)+'\n');
export function removeApprovedLanguageSteps(source){let original=inverseTopicWorkflow(source,".github/workflows/frontend.yml");for(const step of languageWorkflowSteps){if(original.split(step).length!==2)throw Error('language step missing, duplicated or changed');original=original.replace(step,'');}return original;}
