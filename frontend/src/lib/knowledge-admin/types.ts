import type {components} from '../api/generated';
export type ContentMode=components['schemas']['ManagedContentMode'];
export type Ref=components['schemas']['ManagedRef'];
export type Knowledge=components['schemas']['ManagedKnowledge'];
export type KnowledgeSummary=components['schemas']['ManagedKnowledgeSummary'];
export type KnowledgePage=components['schemas']['ManagedKnowledgePage'];
export type CurrentInput=components['schemas']['ManagedCurrentInput'];
export type PublicKnowledge=components['schemas']['ManagedPublicKnowledge'];
export type PublicPage=components['schemas']['ManagedPublicKnowledgePage'];
export type Topic=components['schemas']['ManagedCurrentTopic'];
export type TopicPage=components['schemas']['ManagedTopicPage'];
export type Preview=components['schemas']['ManagedPreview'];
export type Receipt=components['schemas']['ManagedReceipt'];
export type ApplyInput=components['schemas']['ManagedApplyInput'];
export type ImportStatus=components['schemas']['ManagedImportStatus'];
export type StudyDetail=components['schemas']['ManagedStudyDetail'];
export type StudyPage=components['schemas']['ManagedStudyPage'];
export type Overview=components['schemas']['ManagedOverview'];
export type ProgressPage=components['schemas']['ManagedProgressPage'];
export type StudyInput=components['schemas']['ManagedStudyInput'];
export type Note=components['schemas']['ManagedNote'];
export type NoteInput=components['schemas']['ManagedNoteInput'];
export type HistoryPage=components['schemas']['ManagedHistoryPage'];
export type Result<T>={ok:true;data:T}|{ok:false;status:number;code:string};
export class KnowledgeError extends Error{constructor(public readonly code:string,public readonly status=503){super(code);this.name='KnowledgeError'}}

export type ManagedHistoryNumberedPage=import('../api/generated').components['schemas']['ManagedHistoryNumberedPage'];

export type PublishSelection=components['schemas']['KnowledgePublishSelection'];
export type PublishCreateInput=components['schemas']['KnowledgePublishCreateInput'];
export type PublishJob=components['schemas']['KnowledgePublishJob'];
export type PublishItem=components['schemas']['KnowledgePublishItem'];
export type PublishStepInput=components['schemas']['KnowledgePublishStepInput'];
export type PublishStepReceipt=components['schemas']['KnowledgePublishStepReceipt'];
export type PublishJobPage=components['schemas']['KnowledgePublishJobPage'];
export type PublishJobDetail=components['schemas']['KnowledgePublishJobDetail'];
export type PublishReceiptLookup=components['schemas']['KnowledgePublishReceiptLookup'];
