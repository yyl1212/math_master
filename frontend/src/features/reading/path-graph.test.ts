import { it, expect } from "vitest";
import { buildPathGraph, ContractError } from "./path-graph";
import type { PathView, KnowledgeView } from "@/lib/api/types";
const k = (id: string, prerequisites: string[] = []): KnowledgeView => ({
  knowledge: {
    id,
    version: 1,
    domainIds: ["logic"],
    topicIds: [],
    type: "concept",
    title: id,
    titleZh: "",
    statement: "",
    scope: "",
    system: "",
    proof: "",
    conditions: [],
    objectives: [],
    sources: [],
    relations: prerequisites.map((id) => ({
      kind: "prerequisite",
      target: { id, version: 1 },
    })),
  },
  units: [],
  assets: [],
});
const view = (): PathView => ({
  path: {
    id: "test-path",
    version: 1,
    domainIds: ["logic"],
    title: "Test",
    titleZh: "",
    nodes: ["a", "b", "c"].map((id) => ({ id, version: 1 })),
  },
  knowledge: [k("a"), k("b"), k("c", ["a", "b"])],
});
it("usesOnlyExactPrerequisiteVersions", () => {
  const v = view();
  v.knowledge[0].knowledge.relations.push(
    { kind: "related", target: { id: "c", version: 1 } },
    { kind: "derivation", target: { id: "b", version: 1 } },
  );
  expect(buildPathGraph(v)).toEqual({
    levels: [
      [
        { id: "a", version: 1 },
        { id: "b", version: 1 },
      ],
      [{ id: "c", version: 1 }],
    ],
    edges: [
      { from: { id: "a", version: 1 }, to: { id: "c", version: 1 } },
      { from: { id: "b", version: 1 }, to: { id: "c", version: 1 } },
    ],
  });
  for (const change of [
    (v: PathView) => (v.knowledge[2].knowledge.relations[0].target.version = 2),
    (v: PathView) => v.path.nodes.pop(),
    (v: PathView) => v.path.nodes.push(v.path.nodes[0]),
    (v: PathView) =>
      v.knowledge[0].knowledge.relations.push({
        kind: "prerequisite",
        target: { id: "c", version: 1 },
      }),
  ]) {
    const bad = view();
    change(bad);
    expect(() => buildPathGraph(bad)).toThrow(ContractError);
  }
});
