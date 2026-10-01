import type { components } from "./generated";
export type DomainList = components["schemas"]["DomainList"];
export type DomainSummary = components["schemas"]["DomainSummary"];
export type DomainDetail = components["schemas"]["DomainDetail"];
export type PathView = components["schemas"]["PathView"];
export type KnowledgeView = components["schemas"]["KnowledgeView"];
export type AssetView = components["schemas"]["AssetMetadata"];
export type VersionRef = components["schemas"]["ref"];
export type ApiResult<T> =
  | { ok: true; data: T }
  | {
      ok: false;
      kind: "invalid-query" | "not-found" | "unavailable";
      requestId?: string;
    };
export interface GoClient {
  listDomains(query: {
    q: string;
    limit: number;
    offset: number;
  }): Promise<ApiResult<DomainList>>;
  getDomain(id: string): Promise<ApiResult<DomainDetail>>;
  getPath(id: string): Promise<ApiResult<PathView>>;
  getKnowledge(id: string): Promise<ApiResult<KnowledgeView>>;
}
