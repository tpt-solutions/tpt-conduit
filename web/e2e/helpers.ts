import { test, expect, type Page } from "@playwright/test";

// Default single-tenant basic-auth credentials (cmd/conduit-server defaults).
export const AUTH = { username: "admin", password: "secret" };

// Signs in through the web UI and lands on /tickets. The middleware redirects
// unauthenticated page navigations to /login, so this must run first.
export async function login(page: Page): Promise<void> {
  await page.goto("/");
  await expect(page).toHaveURL(/\/login/);
  await page.fill("#username", AUTH.username);
  await page.fill("#password", AUTH.password);
  await page.click('button[type="submit"]');
  await expect(page).toHaveURL(/\/tickets/);
}

// Opens the new-ticket form, creates a ticket for the given workflow, and
// returns the new ticket id parsed from the resulting URL (/tickets/{id}).
export async function createTicket(
  page: Page,
  opts: { workflow: string; version?: string; title: string }
): Promise<string> {
  await page.goto("/tickets/new");
  await expect(page.getByRole("heading", { name: "New ticket" })).toBeVisible();
  await page.fill("#title", opts.title);
  await page.selectOption("#workflow", opts.workflow);
  if (opts.version) {
    await page.selectOption("#version", opts.version);
  }
  await page.click('button[type="submit"]');
  await expect(page).toHaveURL(/\/tickets\/tkt_/);
  const match = page.url().match(/\/tickets\/(tkt_[^/]+)$/);
  return match ? match[1] : "";
}
