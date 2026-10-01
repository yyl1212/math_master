import { test, expect, fitsViewport } from "./fixtures";

test("publishedFixtureCanBeReadWithMathAndOriginalSvg", async ({
  page,
  request,
  runtime,
  scene,
}, info) => {
  await scene("published");
  const errors: string[] = [];
  page.on("pageerror", (e) => errors.push(e.message));
  await page.goto("/domains/elementary-mathematics");
  await page
    .getByRole("link", { name: /Numbers to fractions and percentages/ })
    .click();
  await expect(page.locator("[data-node-key]")).toHaveCount(10);
  const graph = page.getByRole("region", { name: "Knowledge prerequisites" });
  if (info.project.name === "desktop")
    await expect(graph.locator("svg path")).toHaveCount(12);
  await fitsViewport(page);
  if (!process.env.CI)
    await page.screenshot({
      path: info.outputPath("path.png"),
      fullPage: true,
    });
  await page.locator('[data-node-key="equivalent-fractions@1"] h2 a').click();
  await expect(
    page.getByRole("heading", { name: "Equivalent fractions", exact: true }),
  ).toBeVisible();
  await expect(
    page.getByText("Version 1", { exact: true }).first(),
  ).toBeVisible();
  await expect(page.locator("#conditions")).toBeVisible();
  await expect(page.locator(".katex").first()).toBeVisible();
  const image = page.getByRole("img", { name: "Equal shaded amounts" });
  await image.scrollIntoViewIfNeeded();
  await expect(image).toBeVisible();
  await expect
    .poll(() =>
      image.evaluate(
        (el: HTMLImageElement) => el.complete && el.naturalWidth > 0,
      ),
    )
    .toBe(true);
  await expect(image).toHaveAttribute(
    "src",
    "/api/v1/assets/" + runtime.assetSha,
  );
  await expect(
    page.getByText("Math Master project · LicenseRef-MathMaster-Original"),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Sources & use" }),
  ).toBeVisible();
  const svg = await request.get("/api/v1/assets/" + runtime.assetSha);
  expect(svg.status()).toBe(200);
  expect(svg.headers()["content-type"]).toBe("image/svg+xml");
  expect(svg.headers()["cache-control"]).toBe("no-store");
  const head = await request.head("/api/v1/assets/" + runtime.assetSha);
  expect(head.status()).toBe(200);
  expect((await head.body()).length).toBe(0);
  await fitsViewport(page);
  await page.evaluate(() => window.scrollTo(0, 0));
  if (!process.env.CI)
    await page.screenshot({
      path: info.outputPath("knowledge.png"),
      fullPage: true,
    });
  expect(errors).toEqual([]);
});

test("withdrawalStopsNewReadsAndAssetRequests", async ({
  page,
  request,
  runtime,
  scene,
}) => {
  for (const withdrawal of [
    "withdraw-asset",
    "withdraw-unit",
    "withdraw-knowledge",
    "withdraw-prerequisite",
    "withdraw-snapshot",
  ]) {
    await scene("published");
    expect(
      (await request.get("/api/v1/assets/" + runtime.assetSha)).status(),
    ).toBe(200);
    await page.goto("/paths/" + runtime.pathId);
    await expect(page.locator("[data-node-key]")).toHaveCount(10);
    await expect(
      page.locator('section[aria-label="Knowledge prerequisites"] svg path'),
    ).toHaveCount(12);
    await page.locator('[data-node-key="equivalent-fractions@1"] h2 a').click();
    await expect(
      page.getByRole("heading", { name: "Equivalent fractions", exact: true }),
    ).toBeVisible();
    await expect(page.locator("#explanations img")).toHaveCount(1);
    await scene(withdrawal);
    expect(
      (await request.get("/api/v1/assets/" + runtime.assetSha)).status(),
    ).toBe(404);
    await page.reload();
    if (withdrawal === "withdraw-unit" || withdrawal === "withdraw-asset") {
      await expect(
        page.getByRole("heading", {
          name: "Equivalent fractions",
          exact: true,
        }),
      ).toBeVisible();
      await expect(page.locator("img")).toHaveCount(0);
    } else {
      await expect(
        page.getByText("This content is not available."),
      ).toBeVisible();
      expect(
        (await request.get("/api/v1/paths/" + runtime.pathId)).status(),
      ).toBe(404);
    }
    const navigation = page.waitForResponse(
      (r) =>
        new URL(r.url()).pathname === "/knowledge" &&
        r.request().method() === "GET",
    );
    await page
      .getByRole("navigation", { name: "Main navigation" })
      .getByRole("link", { name: "Knowledge Map", exact: true })
      .click();
    expect((await navigation).status()).toBe(200);
    await expect(page.getByLabel("Search learning domains")).toBeVisible();
    await page
      .getByRole("link", { name: "Elementary Mathematics", exact: true })
      .click();
    await expect(
      page.getByRole("heading", { name: "Topics to explore" }),
    ).toBeVisible();
    if (withdrawal !== "withdraw-unit" && withdrawal !== "withdraw-asset") {
      await expect(
        page.getByText("Learning paths are in development."),
      ).toBeVisible();
      expect(
        (
          await request.get("/api/v1/knowledge/" + runtime.knowledgeId)
        ).status(),
      ).toBe(404);
    }
  }
});

test("longNamesAndMultiPrerequisitesFitBothViewports", async ({
  page,
  runtime,
  scene,
}, info) => {
  await scene("long");
  await page.goto("/paths/" + runtime.pathId);
  await expect(page.locator("[data-node-key]")).toHaveCount(10);
  const node = page.locator('[data-node-key="equivalent-fractions@2"]');
  await expect(node.locator("li")).toHaveCount(2);
  await expect(node.locator("h2")).toContainText(
    "Understanding exact assumptions",
  );
  await fitsViewport(page);
  if (info.project.name === "desktop") {
    await expect(
      page.locator('section[aria-label="Knowledge prerequisites"] svg path'),
    ).toHaveCount(12);
    await page.setViewportSize({ width: 1000, height: 900 });
    await fitsViewport(page);
    await expect
      .poll(() =>
        page
          .locator('section[aria-label="Knowledge prerequisites"] svg')
          .getAttribute("viewBox"),
      )
      .not.toBe("0 0 1 1");
  }
  await page.goto("/knowledge/" + runtime.knowledgeId);
  await expect(page.locator(".katex")).toHaveCount(3);
  await fitsViewport(page);
});

test("clientResponsesDoNotExposeInternalConfiguration", async ({
  page,
  request,
  runtime,
  scene,
}) => {
  await scene("published");
  for (const path of [
    "/",
    "/knowledge",
    "/domains/elementary-mathematics",
    "/paths/" + runtime.pathId,
    "/knowledge/" + runtime.knowledgeId,
    "/api/v1/knowledge/" + runtime.knowledgeId,
  ]) {
    const r = await request.get(path);
    expect(r.status()).toBe(200);
    const body = await r.text();
    for (const forbidden of [
      runtime.apiURL,
      runtime.controlURL,
      runtime.token,
      "GO_API_INTERNAL_URL",
      "DATABASE_URL",
      "server.local.json",
      "Knowledge_JSON",
      "assetRoot",
      "postgres://",
      "math_master_test_",
    ])
      expect(body.includes(forbidden)).toBe(false);
  }
  expect(
    (await request.post("/api/v1/knowledge/" + runtime.knowledgeId)).status(),
  ).toBe(405);
  expect((await request.get("/api/v1/private-export")).status()).toBe(404);
  expect((await request.get("/api/v1/assets/not-a-digest")).status()).toBe(404);
  await page.goto("/knowledge/" + runtime.knowledgeId);
  await expect(
    page.locator('script[src*="http:"],iframe,[onclick]'),
  ).toHaveCount(0);
});
