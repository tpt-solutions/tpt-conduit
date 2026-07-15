import { test, expect } from "@playwright/test";
import { login, createTicket } from "./helpers";

// The generic-approval-chain workflow's `review` step is a 3-link approval
// (manager -> director -> finance). The example chain defines roles without
// assigned users, so the engine authorizes a decision when "Decided by" matches
// the required role name. We approve each link in turn and expect the run to
// complete once the full chain is granted.
async function approveLink(page: import("@playwright/test").Page, role: string) {
  await page.fill("#by-review", role);
  await page.click('button:has-text("Approve")');
}

test("approving every link in the chain completes the run", async ({ page }) => {
  await login(page);
  await createTicket(page, {
    workflow: "generic-approval-chain",
    title: `Approval e2e ${Date.now()}`,
  });

  // Jump to the run from the ticket detail page.
  await page.locator('a[href^="/runs/"]').first().click();
  await expect(page).toHaveURL(/\/runs\//);

  // The review approval is pending and actionable.
  await expect(page.getByRole("button", { name: "Approve" })).toBeVisible();
  await expect(page.getByText("review", { exact: true })).toBeVisible();

  for (const role of ["manager", "director", "finance"]) {
    await approveLink(page, role);
    // Each decision triggers a router.refresh(); wait for it to settle before
    // the next action so we don't race the re-render.
    await page.waitForLoadState("networkidle");
  }

  // Once the full chain is granted the run advances through the remaining
  // (no-op) tasks and reaches COMPLETED.
  await expect(page.getByText("COMPLETED", { exact: true }).first()).toBeVisible({ timeout: 20_000 });
});

test("rejecting an approval fails the run", async ({ page }) => {
  await login(page);
  await createTicket(page, {
    workflow: "generic-approval-chain",
    title: `Reject e2e ${Date.now()}`,
  });

  await page.locator('a[href^="/runs/"]').first().click();
  await expect(page).toHaveURL(/\/runs\//);

  await page.fill("#by-review", "manager");
  await page.fill("#comment-review", "not in budget");
  await page.click('button:has-text("Reject")');

  // A rejection fails the run outright.
  await expect(page.getByText("FAILED", { exact: true }).first()).toBeVisible({ timeout: 20_000 });
  await expect(page.getByText("rejected: not in budget", { exact: true })).toBeVisible();
});
