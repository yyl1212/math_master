export type CatalogueStatus = "all" | "planned" | "published";
export function parseCatalogueQuery(
  params: Record<string, string | string[] | undefined>,
): { ok: true; q: string; status: CatalogueStatus } | { ok: false } {
  for (const [key, value] of Object.entries(params)) {
    if (!["q", "status"].includes(key) || Array.isArray(value))
      return { ok: false };
  }
  const q = params.q ?? "",
    status = params.status ?? "all";
  if (
    typeof q !== "string" ||
    typeof status !== "string" ||
    !["all", "planned", "published"].includes(status) ||
    new TextEncoder().encode(q).length > 512
  )
    return { ok: false };
  for (const ch of q) {
    const cp = ch.codePointAt(0)!;
    if (cp < 32 || cp === 127 || (cp >= 0xd800 && cp <= 0xdfff))
      return { ok: false };
  }
  return { ok: true, q, status: status as CatalogueStatus };
}
