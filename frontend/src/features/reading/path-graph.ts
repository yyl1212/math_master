import type { PathView, VersionRef } from "@/lib/api/types";
export class ContractError extends Error {
  constructor() {
    super("Invalid learning path data.");
  }
}
export const refKey = (ref: VersionRef) => ref.id + "@" + ref.version;
export function buildPathGraph(view: PathView): {
  levels: VersionRef[][];
  edges: { from: VersionRef; to: VersionRef }[];
} {
  const nodes = view.path.nodes,
    byId = new Map(nodes.map((n) => [n.id, n])),
    byRef = new Map(nodes.map((n) => [refKey(n), n]));
  const knowledge = new Map(
    view.knowledge.map((v) => [refKey(v.knowledge), v.knowledge]),
  );
  if (
    byId.size !== nodes.length ||
    knowledge.size !== nodes.length ||
    view.knowledge.length !== nodes.length
  )
    throw new ContractError();
  const degrees = new Map(nodes.map((n) => [refKey(n), 0])),
    out = new Map(nodes.map((n) => [refKey(n), [] as string[]])),
    edges: { from: VersionRef; to: VersionRef }[] = [],
    seen = new Set<string>();
  for (const to of nodes) {
    const k = knowledge.get(refKey(to));
    if (!k) throw new ContractError();
    for (const r of k.relations) {
      if (r.kind !== "prerequisite") continue;
      const from = byRef.get(refKey(r.target));
      if (!from) throw new ContractError();
      const edge = refKey(from) + ">" + refKey(to);
      if (seen.has(edge)) continue;
      seen.add(edge);
      edges.push({ from, to });
      degrees.set(refKey(to), degrees.get(refKey(to))! + 1);
      out.get(refKey(from))!.push(refKey(to));
    }
  }
  const order = new Map(nodes.map((n, i) => [refKey(n), i]));
  let layer = nodes.filter((n) => degrees.get(refKey(n)) === 0).map(refKey),
    visited = 0;
  const levels: VersionRef[][] = [];
  while (layer.length) {
    levels.push(layer.map((key) => byRef.get(key)!));
    visited += layer.length;
    const next: string[] = [];
    for (const from of layer)
      for (const to of out.get(from)!) {
        const degree = degrees.get(to)! - 1;
        degrees.set(to, degree);
        if (degree === 0) next.push(to);
      }
    layer = next.sort((a, b) => order.get(a)! - order.get(b)!);
  }
  if (visited !== nodes.length) throw new ContractError();
  return { levels, edges };
}
