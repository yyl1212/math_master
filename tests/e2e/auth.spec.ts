import { test, expect, fitsViewport, signIn, safeScreenshot, TEST_PASSWORD, NEW_PASSWORD } from "./fixtures";

test("registrationLoginAndSignOutUseRealIdentity", async ({ page, scene }, info) => {
  await scene("auth");
  await page.goto("/register");
  await page.getByLabel("Username", { exact: true }).fill("new_learner");
  await page.getByLabel("Password", { exact: true }).fill(TEST_PASSWORD);
  await page.getByRole("button", { name: "Create account", exact: true }).click();
  await expect(page).toHaveURL(/\/login$/);
  await page.getByLabel("Username", { exact: true }).fill("new_learner");
  await page.getByLabel("Password", { exact: true }).fill(NEW_PASSWORD);
  await page.getByRole("button", { name: "Sign in", exact: true }).click();
  await expect(page.getByRole("alert").filter({ hasText: "Invalid username or password." })).toHaveText("Invalid username or password.");
  await expect(page.getByRole("alert").filter({ hasText: "Invalid username or password." })).toBeFocused();
  await expect(page.getByLabel("Password", { exact: true })).toHaveValue("");
  await page.getByLabel("Username", { exact: true }).focus();
  await page.keyboard.press("Tab");
  await expect(page.getByLabel("Password", { exact: true })).toBeFocused();
  await fitsViewport(page);
  await safeScreenshot(page, info, "sign-in");
  await signIn(page, "new_learner", TEST_PASSWORD);
  await expect(page.getByRole("heading", { name: "new_learner", exact: true })).toBeVisible();
  await expect(page.getByRole("link", { name: "Manage users" })).toHaveCount(0);
  await page.getByLabel("Current password").focus(); await page.keyboard.press("Tab");
  await expect(page.getByLabel("New password")).toBeFocused();
  await fitsViewport(page); await safeScreenshot(page, info, "account");
  await page.getByRole("button", { name: "Sign out", exact: true }).click();
  await expect(page).toHaveURL("/");
  await page.goto("/account");
  await expect(page.getByRole("heading", { name: "Sign in to view your account" })).toBeVisible();
});

test("passwordChangeRevokesAnotherBrowserSession", async ({ page, browser, scene, baseURL }) => {
  await scene("auth");
  const other = await browser.newContext({ baseURL, viewport: page.viewportSize()! });
  try {
    const second = await other.newPage();
    await signIn(page, "auth_learner", TEST_PASSWORD);
    await signIn(second, "auth_learner", TEST_PASSWORD);
    await page.getByLabel("Current password").fill(TEST_PASSWORD);
    await page.getByLabel("New password").fill(NEW_PASSWORD);
    await page.getByRole("button", { name: "Change password", exact: true }).click();
    await expect(page).toHaveURL(/\/login$/);
    await second.reload();
    await expect(second.getByRole("heading", { name: "Sign in to view your account" })).toBeVisible();
    await signIn(page, "auth_learner", NEW_PASSWORD);
    await expect(page.getByRole("heading", { name: "auth_learner", exact: true })).toBeVisible();
  } finally { await other.close(); }
});

test("administratorVerifiesRolesAndResetsVerifiedOwner", async ({ page, browser, scene, baseURL }, info) => {
  await scene("auth");
  const actor = await browser.newContext({ baseURL, viewport: page.viewportSize()! });
  try {
    const admin = await actor.newPage();
    await signIn(page, "auth_learner", TEST_PASSWORD);
    await signIn(admin, "auth_admin", TEST_PASSWORD);
    await admin.goto("/admin/users");
    await admin.getByRole("button", { name: "auth_learner learner", exact: true }).click();
    await admin.getByLabel("editor", { exact: true }).check();
    await admin.getByLabel("reviewer", { exact: true }).check();
    await admin.getByLabel("Reason", { exact: true }).fill("Verified editorial training completed");
    await admin.getByRole("button", { name: "Save roles" }).click();
    await expect(admin.getByRole("dialog")).toBeVisible();
    await expect(admin.getByLabel("Your password", { exact: true })).toBeFocused();
    await admin.keyboard.press("Shift+Tab"); await expect(admin.getByRole("button", { name: "Cancel", exact: true })).toBeFocused();
    await admin.keyboard.press("Tab"); await expect(admin.getByLabel("Your password", { exact: true })).toBeFocused();
    await admin.getByLabel("Your password", { exact: true }).fill(TEST_PASSWORD);
    await admin.getByRole("button", { name: "Verify password", exact: true }).click();
    await expect(admin.getByText("Password verified. Submit your change again.")).toBeVisible();
    await page.reload();
    await expect(page.getByRole("heading", { name: "auth_learner", exact: true })).toBeVisible();
    await admin.getByRole("button", { name: "Save roles" }).click();
    await expect(admin.getByText("Changes saved.")).toBeVisible();
    await fitsViewport(admin); await safeScreenshot(admin, info, "administrator");
    await page.reload();
    await expect(page.getByRole("heading", { name: "Sign in to view your account" })).toBeVisible();
    await signIn(page, "auth_learner", TEST_PASSWORD);
    await expect(page.getByText("learner · editor · reviewer", { exact: true })).toBeVisible();
    await page.goto("/admin/users");
    await expect(page.getByRole("heading", { name: "You do not have permission." })).toBeVisible();
    await admin.getByLabel("Reason", { exact: true }).fill("Owner verified through a manual call");
    await admin.getByLabel("Ownership verification").fill("Identity confirmed using offline evidence");
    await admin.getByLabel("Temporary password").fill(NEW_PASSWORD);
    await admin.getByRole("button", { name: "Reset password", exact: true }).click();
    await expect(admin.getByText("Changes saved.")).toBeVisible();
    await signIn(page, "auth_learner", NEW_PASSWORD);
    await expect(page.getByText("Change your password to continue", { exact: true })).toBeVisible();
    await expect(page.getByRole("button", { name: "Sign out everywhere" })).toHaveCount(0);
    await page.getByLabel("Current password").fill(NEW_PASSWORD);
    await page.getByLabel("New password").fill(TEST_PASSWORD);
    await page.getByRole("button", { name: "Change password", exact: true }).click();
    await expect(page).toHaveURL(/\/login$/);
    await signIn(page, "auth_learner", TEST_PASSWORD);
    await expect(page.getByText("Change your password to continue", { exact: true })).toHaveCount(0);
  } finally { await actor.close(); }
});
