import { test, expect, signIn, TEST_PASSWORD } from "./fixtures";

test("csrfCrossOriginAndForgedRolesCannotChangeIdentity", async ({ page, scene }) => {
  await scene("auth");
  await page.goto("/login");
  // Only statuses and booleans leave the browser; secrets never enter diagnostics.
  const statuses = await page.evaluate(async () => {
    const context = await fetch("/api/v1/auth/context", { headers: { "X-Requested-With": "MathMaster" } }).then(r => r.json());
    const post = async (headers: Record<string, string>, body: string) => (await fetch("/api/v1/auth/login", { method: "POST", headers: { "Content-Type": "application/json", ...headers }, body })).status;
    const missing = await post({}, JSON.stringify({ username: "auth_learner", password: "Test-only 中文数学密码 with spaces" }));
    const forged = await post({ "X-CSRF-Token": context.data.csrfToken }, JSON.stringify({ username: "auth_learner", password: "Test-only 中文数学密码 with spaces", roles: ["admin"] }));
    const wrong = await post({ "X-CSRF-Token": "A".repeat(43) }, JSON.stringify({ username: "auth_learner", password: "Test-only 中文数学密码 with spaces" }));
    const session = await fetch("/api/v1/auth/session").then(r => r.json());
    return { missing, forged, wrong, anonymous: session.data.user === null };
  });
  expect(statuses).toEqual({ missing: 403, forged: 400, wrong: 403, anonymous: true });
  await page.goto("/account");
  await expect(page.getByRole("heading", { name: "Sign in to view your account" })).toBeVisible();
});

test("crossSiteWriteIsRejectedAndFailureRecoveryRetainsSession", async ({ page, request, scene }) => {
  await scene("auth");
  await signIn(page, "auth_learner", TEST_PASSWORD);
  const cross = await request.post("/api/v1/auth/logout", { headers: { Origin: "https://foreign.example", "Content-Type": "application/json" }, data: {} });
  expect(cross.status()).toBe(403);
  await scene("auth-unavailable");
  await page.reload();
  await expect(page.getByRole("heading", { name: "Accounts are temporarily unavailable." })).toBeVisible();
  await expect(page.getByRole("link", { name: "Accounts unavailable" })).toBeVisible();
  await scene("auth-recover");
  await page.getByRole("button", { name: "Try again" }).click();
  await expect(page.getByRole("heading", { name: "auth_learner", exact: true })).toBeVisible();
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
  await scene("published");
  await page.goto("/knowledge/equivalent-fractions");
  await expect(page.getByRole("heading", { name: "Equivalent fractions", exact: true })).toBeVisible();
  await expect(page.locator(".katex").first()).toBeVisible();
});
