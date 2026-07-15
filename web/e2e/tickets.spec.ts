import { test, expect } from "@playwright/test";
import { login, createTicket } from "./helpers";

// Core flow: create a ticket and see it reflected in the list and detail views.
test("creating a ticket shows it in the list and detail", async ({ page }) => {
  await login(page);
  const title = `IT helpdesk ${Date.now()}`;
  const id = await createTicket(page, { workflow: "it-helpdesk", title });

  // Detail view.
  await expect(page.getByRole("heading", { name: title })).toBeVisible();
  await expect(page.getByText(/it-helpdesk@/)).toBeVisible();

  // Back to the list; the new ticket should be present.
  await page.goto("/tickets");
  await expect(page.getByRole("link", { name: title })).toBeVisible();
  expect(id).toMatch(/^tkt_/);
});

// Core flow: a created ticket auto-starts a workflow run.
test("a created ticket has a workflow run", async ({ page }) => {
  await login(page);
  await createTicket(page, { workflow: "it-helpdesk", title: `Run check ${Date.now()}` });

  await expect(page.getByRole("heading", { name: "Workflow runs" })).toBeVisible();
  const runLink = page.locator('a[href^="/runs/"]').first();
  await expect(runLink).toBeVisible();
  await runLink.click();
  await expect(page).toHaveURL(/\/runs\//);
});
