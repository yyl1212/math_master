import type { components } from "../api/generated";
type ContentSchema<K extends keyof components["schemas"]> = components["schemas"][K];
export type DraftInput = ContentSchema<"ContentDraftInput">;
export type SaveDraftInput = ContentSchema<"ContentSaveDraftInput">;
export type DraftView = ContentSchema<"ContentDraftView">;
export type DraftSummary = ContentSchema<"ContentDraftSummary">;
export type DraftPage = ContentSchema<"ContentDraftPage">;
export type SubmissionView = ContentSchema<"ContentSubmissionView">;
export type SubmissionSummary = ContentSchema<"ContentSubmissionSummary">;
export type SubmissionPage = ContentSchema<"ContentSubmissionPage">;
export type PublicationView = ContentSchema<"ContentPublicationView">;
export type PublicationPage = ContentSchema<"ContentPublicationPage">;
export type GateReport = ContentSchema<"ContentGateReport">;
export type Package = ContentSchema<"ContentPackage">;
export type Knowledge = ContentSchema<"ContentKnowledge">;
export type Unit = ContentSchema<"ContentUnit">;
export type Path = ContentSchema<"ContentPath">;
export type Source = ContentSchema<"ContentSource">;
export type SourceLink = ContentSchema<"ContentSourceLink">;
export type Asset = ContentSchema<"ContentAsset">;
export type AssetInput = ContentSchema<"ContentAssetInput">;
export type AssetView = ContentSchema<"ContentAssetView">;
export type ReviewInput = ContentSchema<"ContentReviewInput">;
export type ReviewChecks = ContentSchema<"ContentReviewChecks">;
export type AdoptInput = ContentSchema<"ContentAdoptInput">;
export type PrepareInput = ContentSchema<"ContentPrepareInput">;
export type ActivateInput = ContentSchema<"ContentActivateInput">;
export type WithdrawalTarget = ContentSchema<"ContentWithdrawalTarget">;
export type WithdrawalInput = ContentSchema<"ContentWithdrawalInput">;
export type WithdrawalPreview = ContentSchema<"ContentWithdrawalPreview">;
export type WithdrawalResult = ContentSchema<"ContentWithdrawalResult">;
export type Diff = ContentSchema<"ContentDiff">;
export type ContentErrorCode = ContentSchema<"ContentErrorCode">;
export type ContentResult<T> = {
    ok: true;
    data: T;
} | {
    ok: false;
    status: number;
    code: ContentErrorCode;
    message: string;
    requestId: string;
    retryAfter?: number;
};
export type AssetScope = {
    kind: "draft" | "submission";
    id: string;
};
export type ContentListQuery = {
    scope?: "mine" | "review" | "all";
    status?: "editing" | "submitted" | "pending" | "approved" | "returned" | "draft" | "published";
    limit?: number;
    offset?: number;
};
export type ContentRoute = {
    kind: "listDrafts" | "listSubmissions" | "listPublications";
    query?: ContentListQuery;
} | {
    kind: "createDraft" | "adoptDraft" | "prepareRelease" | "previewWithdrawal" | "withdrawVersion";
} | {
    kind: "readDraft" | "saveDraft" | "validateDraft" | "submitDraft" | "readSubmission" | "reviseSubmission" | "decideReview" | "readPublication" | "activateRelease";
    id: string;
} | {
    kind: "readDraftAsset" | "readSubmissionAsset";
    id: string;
    sha: string;
};
export type ContentEndpoint = ContentRoute["kind"];
