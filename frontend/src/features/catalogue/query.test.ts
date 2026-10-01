import { it, expect } from "vitest";
import { parseCatalogueQuery } from "./query";
it("validatesUtf8QueryAndSingletonParameters", () => {
  expect(parseCatalogueQuery({ q: "数".repeat(170) })).toEqual({
    ok: true,
    q: "数".repeat(170),
    status: "all",
  });
  expect(parseCatalogueQuery({ q: "数".repeat(171) })).toEqual({ ok: false });
  for (const params of [
    { q: ["Markov", "概率"] },
    { status: ["all", "planned"] },
    { q: "a\u0000b" },
    { q: "line\nfeed" },
    { status: "locked" },
    { limit: "0" },
    { host: "evil" },
  ])
    expect(parseCatalogueQuery(params)).toEqual({ ok: false });
  expect(parseCatalogueQuery({ q: "%/_", status: "published" })).toEqual({
    ok: true,
    q: "%/_",
    status: "published",
  });
  expect(parseCatalogueQuery({})).toEqual({ ok: true, q: "", status: "all" });
});
