import type {components} from '../api/generated';
export type Head=components['schemas']['NoteShareHead'];
export type OwnNotePage=components['schemas']['OwnNotePage'];
export type OwnNoteSummary=components['schemas']['OwnNoteSummary'];
export type SharedNoteSummary=components['schemas']['SharedNoteSummary'];
export type SharedNoteDetail=components['schemas']['SharedNoteDetail'];
export type SharedNotePage=components['schemas']['SharedNotePage'];
export type LatestSharedNotePage=components['schemas']['LatestSharedNotePage'];
export type AuthenticatedSharedNoteDetail=components['schemas']['AuthenticatedSharedNoteDetail'];
export type AdminNoteSharePage=components['schemas']['AdminNoteSharePage'];
export type AdminNoteShareDetail=components['schemas']['AdminNoteShareDetail'];
export type Receipt=components['schemas']['NoteShareReceipt'];
export type SubmitInput=components['schemas']['NoteShareSubmitInput'];
export type WithdrawInput=components['schemas']['NoteShareWithdrawInput'];
export type DecisionInput=components['schemas']['NoteShareDecisionInput'];
export type NoteResult<T>={ok:true;data:T}|{ok:false;status:number;code:string};
export class NoteShareError extends Error {constructor(public readonly code:string,public readonly status=503){super(code);this.name='NoteShareError'}}

export type OwnNoteNumberedPage=import('../api/generated').components['schemas']['OwnNoteNumberedPage'];
export type SharedNoteNumberedPage=import('../api/generated').components['schemas']['SharedNoteNumberedPage'];
export type AdminNoteShareNumberedPage=import('../api/generated').components['schemas']['AdminNoteShareNumberedPage'];
