import { z } from "zod";
import type {
  DomainList,
  DomainSummary,
  DomainDetail,
  PathView,
  KnowledgeView,
  AssetView,
  VersionRef,
} from "./types";
export const domainSummarySchema = z.object({
  id: z
    .string()
    .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
    .refine((s) => !s.includes("\u0000")),
  order: z.number().int().min(1).max(2147483647),
  name: z.string().refine((s) => !s.includes("\u0000")),
  nameZh: z.string().refine((s) => !s.includes("\u0000")),
  topics: z.array(
    z.object({
      id: z
        .string()
        .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
        .refine((s) => !s.includes("\u0000")),
      name: z.string().refine((s) => !s.includes("\u0000")),
      nameZh: z.string().refine((s) => !s.includes("\u0000")),
    }),
  ),
  relatedDomainIds: z.array(
    z
      .string()
      .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
      .refine((s) => !s.includes("\u0000")),
  ),
  contentStatus: z.enum(["planned", "published"]),
  publishedKnowledgeCount: z.number().int().min(0),
});
export const domainListSchema = z.object({
  items: z.array(domainSummarySchema),
  total: z.number().int().min(0),
  limit: z.number().int().min(1).max(100),
  offset: z.number().int().min(0),
});
export const pathSummarySchema = z.object({
  id: z.string().regex(new RegExp("^[a-z][a-z0-9-]{0,63}$")),
  version: z.number().int().min(1),
  title: z.string(),
  titleZh: z.string(),
});
export const domainDetailSchema = z.object({
  id: z
    .string()
    .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
    .refine((s) => !s.includes("\u0000")),
  order: z.number().int().min(1).max(2147483647),
  name: z.string().refine((s) => !s.includes("\u0000")),
  nameZh: z.string().refine((s) => !s.includes("\u0000")),
  topics: z.array(
    z.object({
      id: z
        .string()
        .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
        .refine((s) => !s.includes("\u0000")),
      name: z.string().refine((s) => !s.includes("\u0000")),
      nameZh: z.string().refine((s) => !s.includes("\u0000")),
    }),
  ),
  relatedDomainIds: z.array(
    z
      .string()
      .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
      .refine((s) => !s.includes("\u0000")),
  ),
  contentStatus: z.enum(["planned", "published"]),
  publishedKnowledgeCount: z.number().int().min(0),
  paths: z.array(pathSummarySchema),
});
export const refSchema = z.object({
  id: z
    .string()
    .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
    .refine((s) => !s.includes("\u0000")),
  version: z.number().int().min(1).max(2147483647),
});
export const pathSchema = z.object({
  id: z
    .string()
    .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
    .refine((s) => !s.includes("\u0000")),
  version: z.number().int().min(1).max(2147483647),
  domainIds: z.array(
    z
      .string()
      .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
      .refine((s) => !s.includes("\u0000")),
  ),
  title: z.string().refine((s) => !s.includes("\u0000")),
  titleZh: z.string().refine((s) => !s.includes("\u0000")),
  nodes: z.array(refSchema),
});
export const sourceSchema = z.object({
  kind: z.enum(["original", "external"]),
  author: z.string().refine((s) => !s.includes("\u0000")),
  title: z.string().refine((s) => !s.includes("\u0000")),
  url: z.string().refine((s) => !s.includes("\u0000")),
  accessedAt: z.string().refine((s) => !s.includes("\u0000")),
  license: z.string().refine((s) => !s.includes("\u0000")),
  attribution: z.string().refine((s) => !s.includes("\u0000")),
});
export const relationSchema = z.object({
  kind: z.enum(["prerequisite", "derivation", "related"]),
  target: refSchema,
});
export const knowledgeSchema = z.object({
  id: z
    .string()
    .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
    .refine((s) => !s.includes("\u0000")),
  version: z.number().int().min(1).max(2147483647),
  domainIds: z.array(
    z
      .string()
      .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
      .refine((s) => !s.includes("\u0000")),
  ),
  topicIds: z.array(
    z
      .string()
      .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
      .refine((s) => !s.includes("\u0000")),
  ),
  type: z.enum([
    "concept",
    "definition",
    "axiom",
    "theorem",
    "corollary",
    "method",
    "mathematical-thinking",
  ]),
  title: z.string().refine((s) => !s.includes("\u0000")),
  titleZh: z.string().refine((s) => !s.includes("\u0000")),
  statement: z.string().refine((s) => !s.includes("\u0000")),
  scope: z.string().refine((s) => !s.includes("\u0000")),
  system: z.string().refine((s) => !s.includes("\u0000")),
  proof: z.string().refine((s) => !s.includes("\u0000")),
  objectives: z.array(z.string().refine((s) => !s.includes("\u0000"))),
  conditions: z.array(z.string().refine((s) => !s.includes("\u0000"))),
  sources: z.array(sourceSchema),
  relations: z.array(relationSchema),
});
export const angleSchema = z.object({
  kind: z.string().refine((s) => !s.includes("\u0000")),
  body: z.string().refine((s) => !s.includes("\u0000")),
});
export const unitSchema = z.object({
  id: z
    .string()
    .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
    .refine((s) => !s.includes("\u0000")),
  version: z.number().int().min(1).max(2147483647),
  knowledge: refSchema,
  angles: z.array(angleSchema),
  examples: z.array(z.string().refine((s) => !s.includes("\u0000"))),
  counterexamples: z.array(z.string().refine((s) => !s.includes("\u0000"))),
  assetIds: z.array(
    z
      .string()
      .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
      .refine((s) => !s.includes("\u0000")),
  ),
});
export const assetMetadataSchema = z.object({
  id: z
    .string()
    .regex(new RegExp("^[a-z][a-z0-9-]{0,63}$"))
    .refine((s) => !s.includes("\u0000")),
  sha256: z
    .string()
    .regex(new RegExp("^[a-f0-9]{64}$"))
    .refine((s) => !s.includes("\u0000")),
  author: z.string().refine((s) => !s.includes("\u0000")),
  license: z.string().refine((s) => !s.includes("\u0000")),
  attribution: z.string().refine((s) => !s.includes("\u0000")),
  knowledge: refSchema,
});
export const knowledgeViewSchema = z.object({
  knowledge: knowledgeSchema,
  units: z.array(unitSchema),
  assets: z.array(assetMetadataSchema),
});
export const pathViewSchema = z.object({
  path: pathSchema,
  knowledge: z.array(knowledgeViewSchema),
});
export const errorSchema = z.object({
  error: z.object({
    code: z.enum([
      "INVALID_QUERY",
      "NOT_FOUND",
      "SERVICE_UNAVAILABLE",
      "INTERNAL_ERROR",
    ]),
    message: z.string(),
    requestId: z.string(),
  }),
});
type Equal<A, B> = [A] extends [B] ? ([B] extends [A] ? true : never) : never;
const contractChecks: [
  Equal<z.infer<typeof domainListSchema>, DomainList>,
  Equal<z.infer<typeof domainSummarySchema>, DomainSummary>,
  Equal<z.infer<typeof domainDetailSchema>, DomainDetail>,
  Equal<z.infer<typeof pathViewSchema>, PathView>,
  Equal<z.infer<typeof knowledgeViewSchema>, KnowledgeView>,
  Equal<z.infer<typeof assetMetadataSchema>, AssetView>,
  Equal<z.infer<typeof refSchema>, VersionRef>,
] = [true, true, true, true, true, true, true];
void contractChecks;
