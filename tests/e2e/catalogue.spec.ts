import { test, expect, fitsViewport } from "./fixtures";

test("draftCatalogueShowsSixteenDomainsAndFiftySixTopics", async ({
  page,
  request,
  runtime,
  scene,
}, info) => {
  await scene("draft");
  const requests: string[] = [];
  page.on("request", (r) => requests.push(r.url()));
  await page.goto("/");
  await expect(
    page.getByRole("heading", { name: "A clear path through mathematics." }),
  ).toBeVisible();
  await expect(page.locator(".domain-card")).toHaveCount(16);
  await expect(
    page.getByText("0 published knowledge points", { exact: true }),
  ).toHaveCount(16);
  await expect(
    page.getByText(
      /\bstreak\b|\bXP\b|\bMastered\b|Your progress|\bdemonstration\b/i,
    ),
  ).toHaveCount(0);
  await fitsViewport(page);
  if (!process.env.CI)
    await page.screenshot({ path: info.outputPath("hub.png"), fullPage: true });
  await page.goto("/knowledge");
  await expect(page.locator(".domain-card")).toHaveCount(16);
  await expect(page.locator(".domain-card li")).toHaveCount(56);
  await expect(page.locator(".status")).toHaveText(
    Array(16).fill("In development"),
  );
  for (const route of [
    "/api/v1/knowledge/" + runtime.knowledgeId,
    "/api/v1/paths/" + runtime.pathId,
    "/api/v1/assets/" + runtime.assetSha,
  ])
    expect((await request.get(route)).status()).toBe(404);
  expect(requests.some((u) => /\/design\/|\/Knowledge_JSON\//.test(u))).toBe(
    false,
  );
  await page
    .getByRole("link", { name: "Elementary Mathematics", exact: true })
    .click();
  await expect(
    page.getByText("Learning paths are in development."),
  ).toBeVisible();
  await expect(
    page.getByRole("heading", { name: "Arithmetic", exact: true }),
  ).toBeVisible();
  await fitsViewport(page);
});

test("englishChineseSearchAndHistoryWork", async ({ page, scene }) => {
  await scene("draft");
  await page.goto("/knowledge");
  const input = page.getByLabel("Search learning domains");
  await input.fill("Markov");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await expect(page.locator(".domain-card")).toHaveCount(1);
  await expect(
    page.getByRole("link", {
      name: "Probability & Stochastic Processes",
      exact: true,
    }),
  ).toBeVisible();
  await input.fill("概率");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await expect(page.locator(".domain-card")).toHaveCount(1);
  await expect(page).toHaveURL(/q=%E6%A6%82%E7%8E%87/);
  await expect(input).toHaveValue("概率");
  await page.goBack();
  await expect(input).toHaveValue("Markov");
  await page.goForward();
  await expect(input).toHaveValue("概率");
  await input.fill("%");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await expect(page.getByText("No domains match your search.")).toBeVisible();
  await page.goto("/knowledge");
  await page.getByLabel("Content status").selectOption("published");
  await page.getByRole("button", { name: "Search", exact: true }).click();
  await expect(page.getByText("No domains match your search.")).toBeVisible();
  await fitsViewport(page);
});

test("emptyAndUnavailableAreDistinct", async ({ page, request, scene }) => {
  await scene("empty");
  await page.goto("/knowledge");
  await expect(
    page.getByText("The catalogue is being prepared."),
  ).toBeVisible();
  await expect(
    page.getByText("Content is temporarily unavailable."),
  ).toHaveCount(0);
  await scene("unavailable");
  await page.reload();
  await expect(
    page.getByText("Content is temporarily unavailable."),
  ).toBeVisible();
  expect((await request.get("/api/v1/domains")).status()).toBe(503);
  await scene("draft");
  await page.getByRole("button", { name: "Try again" }).click();
  await expect(page.locator(".domain-card")).toHaveCount(16);
  await page.goto("/knowledge/missing-node");
  await expect(page.getByText("This content is not available.")).toBeVisible();
  await fitsViewport(page);
});

test("keyboardCanReachEveryDomain", async ({ page, scene }) => {
  await scene("draft");
  await page.goto("/knowledge");
  await expect(page.locator(".domain-card")).toHaveCount(16);
  await expect(page.getByLabel("Search learning domains")).toBeVisible();
  await page.keyboard.press("Tab");
  await expect(
    page.getByRole("link", { name: "Skip to content" }),
  ).toBeFocused();
  const seen = new Set<string>();
  for (let i = 0; i < 60 && seen.size < 16; i++) {
    await page.keyboard.press("Tab");
    const focus = await page.evaluate(() => ({
      href: document.activeElement?.getAttribute("href"),
      outline: getComputedStyle(document.activeElement!).outlineStyle,
    }));
    if (focus.href?.startsWith("/domains/")) {
      seen.add(focus.href);
      expect(focus.outline).toBe("solid");
    }
  }
  expect(seen.size).toBe(16);
  await page.keyboard.press("Enter");
  await expect(
    page.getByRole("heading", { name: "Topics to explore" }),
  ).toBeVisible();
  await fitsViewport(page);
});
