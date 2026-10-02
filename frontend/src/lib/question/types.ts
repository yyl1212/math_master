import type { components } from "../api/generated";
type S<K extends keyof components["schemas"]> = components["schemas"][K];
export type DraftInput = S<"QuestionDraftInput">;
export type SaveDraftInput = S<"QuestionSaveDraftInput">;
export type DraftView = S<"QuestionDraftView">;
export type DraftSummary = S<"QuestionDraftSummary">;
export type DraftPage = S<"QuestionDraftPage">;
export type SubmissionView = S<"QuestionSubmissionView">;
export type SubmissionSummary = S<"QuestionSubmissionSummary">;
export type SubmissionPage = S<"QuestionSubmissionPage">;
export type Instance = S<"QuestionInstance">;
export type InstancePage = S<"QuestionInstancePage">;
export type QuestionPackage = S<"QuestionPackage">;
export type Template = S<"QuestionTemplate">;
export type FixedQuestion = S<"QuestionFixedQuestion">;
export type Blueprint = S<"QuestionBlueprint">;
export type ValidationReport = S<"QuestionValidationReport">;
export type ReviewInput = S<"QuestionReviewInput">;
export type ReviewChecks = S<"QuestionReviewChecks">;
export type PublicationSummary = S<"QuestionPublicationSummary">;
export type PublicationPage = S<"QuestionPublicationPage">;
export type MemberPage = S<"QuestionMemberPage">;
export type ChangePage = S<"QuestionChangePage">;
export type PrepareInput = S<"QuestionPrepareInput">;
export type ActivateInput = S<"QuestionActivateInput">;
export type WithdrawalTarget = S<"QuestionWithdrawalTarget">;
export type WithdrawalInput = S<"QuestionWithdrawalInput">;
export type WithdrawalPreview = S<"QuestionWithdrawalPreview">;
export type WithdrawalResult = S<"QuestionWithdrawalResult">;
export type CoverageNode = S<"QuestionCoverageNode">;
export type CoverageReport = S<"QuestionCoverageReport">;
export type QuestionErrorCode = S<"QuestionErrorCode">;
export type QuestionResult<T> = {
    ok: true;
    data: T;
} | {
    ok: false;
    status: number;
    code: QuestionErrorCode;
    message: string;
    requestId: string;
    retryAfter?: number;
};
export type PageQuery = {
    limit?: number;
    offset?: number;
};
export type QuestionListQuery = PageQuery & {
    scope?: "mine" | "review" | "all";
    status?: "editing" | "submitted" | "pending" | "approved" | "returned" | "prepared" | "published";
};
export type CoverageQuery = PageQuery & {
    knowledgeId?: string;
};
export type QuestionRoute = {
    kind: "listDrafts" | "listSubmissions" | "listPublications";
    query?: QuestionListQuery;
} | {
    kind: "listInstances" | "listMembers" | "listChanges";
    id: string;
    query?: PageQuery;
} | {
    kind: "previewWithdrawal";
    query?: PageQuery;
} | {
    kind: "readCoverage";
    query?: CoverageQuery;
} | {
    kind: "createDraft" | "adoptDraft" | "prepareRelease" | "withdrawVersion";
} | {
    kind: "readDraft" | "saveDraft" | "validateDraft" | "submitDraft" | "readSubmission" | "reviseSubmission" | "decideReview" | "readPublication" | "activateRelease";
    id: string;
};
export type QuestionAction = QuestionRoute["kind"];
