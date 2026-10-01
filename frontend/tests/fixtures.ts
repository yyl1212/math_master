import catalogue from "../../content/catalogue/domains.json";
import type { DomainList } from "../src/lib/api/types";
export function catalogueFixture(): DomainList {
  return {
    items: catalogue.domains.map((d) => ({
      ...d,
      contentStatus: "planned" as const,
      publishedKnowledgeCount: 0,
    })),
    total: 16,
    limit: 100,
    offset: 0,
  };
}
