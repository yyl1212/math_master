import {expect,type Runtime} from "./fixtures";
import {actor,wireContent} from "./content-helpers";
import type {Page,APIRequestContext} from "../../frontend/node_modules/@playwright/test/index.js";
import type {Identity,KnowledgeDetail,AttemptView,ResultView,SubmitInput} from "../../frontend/src/lib/learning/types";
export {actor};export const wireLearning=wireContent;
export const rootKnowledge="learning-root",routeID="learning-route";
export type LearningState={knowledgeHead:string;questionHead:string;knowledge:Identity[];path:Identity;blueprints:Identity[];learningEvents:number;practiceAttempts:number;assessmentAttempts:number;answers:number;results:number;qualifications:number;unlocks:number;exposures:number;views:number};
export async function learningState(request:APIRequestContext,runtime:Runtime):Promise<LearningState>{const r=await request.get(runtime.controlURL+"/learning/state",{headers:{Authorization:"Bearer "+runtime.token}});expect(r.status()).toBe(200);return r.json()}
export async function knowledgeDetail(page:Page,id=rootKnowledge,version=1){const r=await wireLearning<KnowledgeDetail>(page,`/api/v1/learning/knowledge/${id}?version=${version}`);expect(r.status).toBe(200);return r.data}
export function arithmeticAnswer(prompt:string){const match=/Calculate ([0-9]+) \+ ([0-9]+)\./.exec(prompt);if(!match)throw Error("Unsupported original fixture prompt");return String(Number(match[1])+Number(match[2]))}
export function answers(attempt:AttemptView,correct=5):SubmitInput {return {answers:attempt.questions.map((q,i)=>({position:q.position,instance:q.instance,answer:i<correct?{kind:"numeric" as const,raw:arithmeticAnswer(q.prompt)}:{kind:"skipped" as const}}))}}
export async function passedAssessment(page:Page,id=rootKnowledge,mode:"node"|"diagnostic"|"review"="node",correct=5){const d=await knowledgeDetail(page,id);const r=await wireLearning<AttemptView>(page,"/api/v1/learning/assessments","POST",{knowledge:d.state.knowledge,blueprint:d.blueprints[0].blueprint,mode,expectedKnowledgeHead:d.knowledgeHead,expectedQuestionHead:d.questionHead});expect(r.status).toBe(201);expect(JSON.stringify(r.data)).not.toMatch(/correctNumeric|correctChoiceId|explanation/);const result=await wireLearning<ResultView>(page,`/api/v1/learning/assessments/${r.data.summary.id}/submit`,"POST",answers(r.data,correct));expect(result.status).toBe(200);return result.data}
