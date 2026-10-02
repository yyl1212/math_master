import type { components } from "../api/generated";
type S<K extends keyof components["schemas"]> = components["schemas"][K];
export type Identity = S<"LearningIdentity">;
export type InstanceIdentity = S<"LearningInstanceIdentity">;
export type QualificationView = S<"LearningQualificationView">;
export type PrerequisiteState = S<"LearningPrerequisiteState">;
export type KnowledgeState = S<"LearningKnowledgeState">;
export type BlueprintOption = S<"LearningBlueprintOption">;
export type KnowledgeDetail = S<"LearningKnowledgeDetail">;
export type StartInput = S<"LearningStartInput">;
export type CompleteInput = S<"LearningCompleteInput">;
export type EnrollInput = S<"LearningEnrollInput">;
export type PathSummary = S<"LearningPathSummary">;
export type PathNode = S<"LearningPathNode">;
export type PathView = S<"LearningPathView">;
export type PublishedPathSummary = S<"LearningPublishedPathSummary">;
export type HistoryEntry = S<"LearningHistoryEntry">;
export type Overview = S<"LearningOverview">;
export type SafeQuestion = S<"LearningSafeQuestion">;
export type Answer = S<"LearningAnswer">;
export type PracticeAnswerInput = S<"LearningPracticeAnswerInput">;
export type PositionAnswer = S<"LearningPositionAnswer">;
export type SubmitInput = S<"LearningSubmitInput">;
export type PracticeCreateInput = S<"LearningPracticeCreateInput">;
export type CreateInput = S<"LearningCreateInput">;
export type AttemptSummary = S<"LearningAttemptSummary">;
export type AttemptView = S<"LearningAttemptView">;
export type ResultItem = S<"LearningResultItem">;
export type PracticeResult = S<"LearningPracticeResult">;
export type PracticeView = S<"LearningPracticeView">;
export type ProgressUpdate = S<"LearningProgressUpdate">;
export type ResultView = S<"LearningResultView">;
export type LearningErrorCode = S<"LearningErrorCode">;
export type AssetRef = S<"LearningAssetRef">;
export type LearningPage<T> = {
    items: T[];
    total: number;
    limit: number;
    offset: number;
};
export type PageQuery = {
    limit?: number;
    offset?: number;
};
export type LearningRoute = {
    kind: "readOverview" | "createPractice" | "createAssessment";
} | {
    kind: "listKnowledge" | "listPaths" | "listHistory";
    query?: PageQuery;
} | {
    kind: "readKnowledge";
    id: string;
    version: number;
} | {
    kind: "startKnowledge" | "completeKnowledge" | "enrollPath";
    id: string;
} | {
    kind: "readPath" | "readPractice" | "answerPractice" | "revealPractice" | "abandonPractice" | "readAssessment" | "submitAssessment" | "abandonAssessment" | "readAssessmentResult";
    id: string;
} | {
    kind: "listPathNodes";
    id: string;
    query?: PageQuery;
} | {
    kind: "readAsset";
    id: string;
    sha: string;
};
export type LearningAction = LearningRoute["kind"];
export type LearningResult<T> = {
    ok: true;
    data: T;
} | {
    ok: false;
    status: number;
    code: LearningErrorCode;
    message: string;
    requestId: string;
    retryAfter?: number;
    retryAt?: string;
    activeAttempt?: AttemptSummary;
    formatCode?: string;
};
export interface LearningReadClient {
    readLearningOverview(): Promise<LearningResult<Overview>>;
    listLearningKnowledge(query: PageQuery): Promise<LearningResult<LearningPage<KnowledgeState>>>;
    readLearningKnowledge(id: string, version: number): Promise<LearningResult<KnowledgeDetail>>;
    listLearningPaths(query: PageQuery): Promise<LearningResult<LearningPage<PathSummary>>>;
    readLearningPath(id: string): Promise<LearningResult<PathView>>;
    listLearningPathNodes(id: string, query: PageQuery): Promise<LearningResult<LearningPage<PathNode>>>;
    readPractice(id: string): Promise<LearningResult<PracticeView>>;
    readAssessment(id: string): Promise<LearningResult<AttemptView>>;
    readAssessmentResult(id: string): Promise<LearningResult<ResultView>>;
    listLearningHistory(query: PageQuery): Promise<LearningResult<LearningPage<HistoryEntry>>>;
    readLearningAsset(attemptID: string, sha: string): Promise<LearningResult<Uint8Array>>;
}
