import { it, expect } from "vitest";
import { createPublicProxy } from "./public-proxy";
const origin = "http://127.0.0.1:18080",
  sha = "a".repeat(64);
it("onlyForwardsPublicReadRoutes", async () => {
  let calls = 0;
  const proxy = createPublicProxy(origin, async () => {
    calls++;
    return new Response("{}", {
      headers: { "Content-Type": "application/json" },
    });
  });
  for (const segments of [
    ["private"],
    ["domains", ".."],
    ["knowledge", "a/b"],
    ["knowledge", "a%2Fb"],
    ["assets", "ABC"],
    ["assets", sha, "extra"],
  ]) {
    expect(
      (await proxy(new Request("http://site.test/api/v1/private"), segments))
        .status,
    ).toBe(404);
  }
  expect(calls).toBe(0);
  const overriding = await proxy(
    new Request(
      "http://site.test/api/v1/knowledge/numbers?url=https://evil.test",
    ),
    ["knowledge", "numbers"],
  );
  expect(overriding.status).toBe(400);
  expect(calls).toBe(0);
  expect(
    (
      await proxy(
        new Request("http://site.test/api/v1/domains", { method: "POST" }),
        ["domains"],
      )
    ).status,
  ).toBe(405);
});
it("dropsCredentialsAndPreservesNoStore", async () => {
  let target = "",
    options: RequestInit | undefined;
  const proxy = createPublicProxy(origin, async (input, init) => {
    target = String(input);
    options = init;
    return new Response("<svg/>", {
      headers: {
        "Content-Type": "image/svg+xml",
        "X-Content-Type-Options": "nosniff",
        "Cache-Control": "no-store",
        "Content-Security-Policy": "sandbox; default-src 'none'",
        "X-Request-ID": "safe",
        "Set-Cookie": "private=secret",
        "X-Internal": "secret",
      },
    });
  });
  const r = await proxy(
    new Request("http://site.test/api/v1/assets/" + sha, {
      headers: {
        Cookie: "secret",
        Authorization: "secret",
        "X-Request-ID": "client",
      },
    }),
    ["assets", sha],
  );
  expect(target).toBe(origin + "/api/v1/assets/" + sha);
  expect(new Headers(options?.headers).get("Cookie")).toBeNull();
  expect(new Headers(options?.headers).get("Authorization")).toBeNull();
  expect(r.headers.get("Cache-Control")).toBe("no-store");
  expect(r.headers.get("Content-Type")).toBe("image/svg+xml");
  expect(r.headers.get("X-Content-Type-Options")).toBe("nosniff");
  expect(r.headers.get("Set-Cookie")).toBeNull();
  expect(r.headers.get("X-Internal")).toBeNull();
  expect(await r.text()).toBe("<svg/>");
});
it("stopsReadingAtByteLimits", async () => {
  let cancelled = false;
  const stream = new ReadableStream<Uint8Array>({
    pull(c) {
      c.enqueue(new Uint8Array(600000));
    },
    cancel() {
      cancelled = true;
    },
  });
  const proxy = createPublicProxy(
    origin,
    async () =>
      new Response(stream, { headers: { "Content-Type": "image/svg+xml" } }),
  );
  const r = await proxy(new Request("http://site.test/api/v1/assets/" + sha), [
    "assets",
    sha,
  ]);
  expect(r.status).toBe(503);
  expect(cancelled).toBe(true);
});
it("headReturnsNoBody", async () => {
  const proxy = createPublicProxy(
    origin,
    async () =>
      new Response(null, {
        headers: { "Content-Type": "image/svg+xml", "Content-Length": "120" },
      }),
  );
  const r = await proxy(
    new Request("http://site.test/api/v1/assets/" + sha, { method: "HEAD" }),
    ["assets", sha],
  );
  expect(r.status).toBe(200);
  expect(await r.text()).toBe("");
  expect(r.headers.get("Content-Length")).toBe("120");
});
it("redactsUpstreamFailureAndRejectsContentType", async () => {
  for (const response of [
    new Response("database password=secret", { status: 500 }),
    new Response("<script/>", { headers: { "Content-Type": "text/html" } }),
  ]) {
    const proxy = createPublicProxy(origin, async () => response);
    const r = await proxy(new Request("http://site.test/api/v1/domains"), [
      "domains",
    ]);
    expect(r.status).toBe(503);
    expect(await r.text()).not.toContain("secret");
  }
});

it("rejectsMalformedUpstreamErrorsButKeepsValidHeadStatus", async () => {
  for (const status of [400, 404]) {
    const proxy = createPublicProxy(
      origin,
      async () =>
        new Response("{}", {
          status,
          headers: { "Content-Type": "application/json" },
        }),
    );
    expect(
      (
        await proxy(new Request("http://site.test/api/v1/knowledge/numbers"), [
          "knowledge",
          "numbers",
        ])
      ).status,
    ).toBe(503);
  }
  const head = createPublicProxy(
    origin,
    async () =>
      new Response(null, {
        status: 404,
        headers: { "Content-Type": "application/json" },
      }),
  );
  const r = await head(
    new Request("http://site.test/api/v1/assets/" + sha, { method: "HEAD" }),
    ["assets", sha],
  );
  expect(r.status).toBe(404);
  expect(await r.text()).toBe("");
});
