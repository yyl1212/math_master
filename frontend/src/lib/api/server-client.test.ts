import { describe, it, expect, vi } from "vitest";
import { createGoClient } from "./server-client";
import { parseGoOrigin } from "./server-config";
const origin = "http://127.0.0.1:18080";
const list = {
  items: [
    {
      id: "logic",
      order: 1,
      name: "Logic",
      nameZh: "逻辑",
      topics: [],
      relatedDomainIds: [],
      contentStatus: "planned",
      publishedKnowledgeCount: 0,
    },
  ],
  total: 1,
  limit: 100,
  offset: 0,
};
const json = (body: unknown, status = 200) =>
  new Response(JSON.stringify(body), {
    status,
    headers: { "Content-Type": "application/json" },
  });
describe("Go client", () => {
  it("handlesWireContractWithoutFallbackData", async () => {
    const client = createGoClient(origin, async () => json(list));
    expect(await client.listDomains({ q: "", limit: 100, offset: 0 })).toEqual({
      ok: true,
      data: list,
    });
    for (const [response, kind] of [
      [
        json(
          {
            error: {
              code: "NOT_FOUND",
              message: "Resource not found.",
              requestId: "safe-id",
            },
          },
          404,
        ),
        "not-found",
      ],
      [json({}, 503), "unavailable"],
      [
        new Response("invalid-json", {
          headers: { "Content-Type": "application/json" },
        }),
        "unavailable",
      ],
      [json({ data: { id: "incomplete" } }), "unavailable"],
    ] as const) {
      const result = await createGoClient(
        origin,
        async () => response,
      ).getKnowledge("numbers");
      expect(result).toMatchObject({ ok: false, kind });
      expect(JSON.stringify(result)).not.toContain(origin);
    }
  });
  it("validatesCompleteNestedFields", async () => {
    const bad = {
      ...list,
      items: [{ ...list.items[0], topics: [{ id: "sets", name: "Sets" }] }],
    };
    expect(
      await createGoClient(origin, async () => json(bad)).listDomains({
        q: "",
        limit: 100,
        offset: 0,
      }),
    ).toMatchObject({ ok: false, kind: "unavailable" });
  });
  it("abortsAfterFiveSeconds", async () => {
    vi.useFakeTimers();
    const fetcher: typeof fetch = async (_input, init) =>
      new Promise((_resolve, reject) =>
        init?.signal?.addEventListener("abort", () =>
          reject(new Error("aborted")),
        ),
      );
    let settled = false;
    const pending = createGoClient(origin, fetcher).getKnowledge("numbers");
    pending.then(() => {
      settled = true;
    });
    await vi.advanceTimersByTimeAsync(4999);
    expect(settled).toBe(false);
    await vi.advanceTimersByTimeAsync(1);
    expect(await pending).toMatchObject({ ok: false, kind: "unavailable" });
  });
  it("neverFollowsRedirectsOrPublishesOrigin", async () => {
    let target = "",
      options: RequestInit | undefined;
    const fetcher: typeof fetch = async (input, init) => {
      target = String(input);
      options = init;
      return json(list);
    };
    const result = await createGoClient(origin, fetcher).listDomains({
      q: "概率 & %",
      limit: 100,
      offset: 0,
    });
    expect(new URL(target).searchParams.get("q")).toBe("概率 & %");
    expect(options).toMatchObject({ cache: "no-store", redirect: "error" });
    expect(JSON.stringify(result)).not.toContain(origin);
    expect(
      await createGoClient(origin, fetcher).getKnowledge("../private"),
    ).toMatchObject({ ok: false, kind: "not-found" });
  });
  it("rejectsNonOriginPrivateConfiguration", () => {
    expect(parseGoOrigin(origin)).toBe(origin);
    for (const bad of [
      "ftp://localhost",
      "http://user:secret@localhost",
      "http://localhost/api",
      "http://localhost/?url=x",
      "http://localhost/#x",
    ])
      expect(() => parseGoOrigin(bad)).toThrow();
  });
});

it("treatsMalformedErrorBodiesAsUnavailable", async () => {
  for (const status of [400, 404]) {
    const r = await createGoClient(origin, async () =>
      json({}, status),
    ).getKnowledge("numbers");
    expect(r).toEqual({ ok: false, kind: "unavailable" });
  }
});
