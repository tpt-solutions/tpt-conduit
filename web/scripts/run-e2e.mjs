// End-to-end runner for the web app (TODO.md cross-cutting e2e task).
//
// It builds and starts the Go conduit-server (in-memory) on :8080, waits for it
// to accept GraphQL traffic, then delegates to Playwright, which starts the
// Next.js dev server itself (see playwright.config.ts webServer). The backend
// is torn down on exit.
//
// Usage:  npm run e2e            (from web/)
// Prereq: `go` on PATH, and `npx playwright install chromium` once.

import { spawn, spawnSync } from "node:child_process";
import { fileURLToPath } from "node:url";
import { dirname, resolve } from "node:path";
import { existsSync, mkdtempSync } from "node:fs";
import { tmpdir } from "node:os";

const webDir = resolve(dirname(fileURLToPath(import.meta.url)), "..");
const repoRoot = resolve(webDir, "..");
const API_URL = process.env.CONDUIT_API_URL || "http://localhost:8080";

const isWin = process.platform === "win32";
const binName = isWin ? "conduit-server.exe" : "conduit-server";
const binPath = resolve(mkdtempSync(resolve(tmpdir(), "conduit-")), binName);

function fail(msg) {
  console.error(`e2e: ${msg}`);
  process.exitCode = 1;
}

// 1. Build the backend binary.
console.log("e2e: building conduit-server ...");
const build = spawnSync("go", ["build", "-o", binPath, "./cmd/conduit-server"], {
  cwd: repoRoot,
  stdio: "inherit",
});
if (build.status !== 0) {
  fail("go build failed");
  process.exit(1);
}

// 2. Start the backend.
const backend = spawn(binPath, [], {
  cwd: repoRoot,
  env: {
    ...process.env,
    CONDUIT_ADDR: ":8080",
    CONDUIT_USERNAME: "admin",
    CONDUIT_PASSWORD: "secret",
  },
  stdio: "inherit",
});

let backendDown = false;
function stopBackend() {
  if (backendDown || backend.killed) return;
  backendDown = true;
  try {
    backend.kill(isWin ? "SIGTERM" : "SIGTERM");
  } catch {
    /* ignore */
  }
}
process.on("exit", stopBackend);
process.on("SIGINT", () => {
  stopBackend();
  process.exit(130);
});
process.on("SIGTERM", () => {
  stopBackend();
  process.exit(143);
});

// 3. Wait for the backend to accept GraphQL requests.
const AUTH_HEADER =
  "Basic " + Buffer.from("admin:secret").toString("base64");

async function waitForBackend(timeoutMs = 30_000) {
  const start = Date.now();
  while (Date.now() - start < timeoutMs) {
    try {
      const res = await fetch(API_URL + "/graphql", {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: AUTH_HEADER,
        },
        body: JSON.stringify({ query: "{ workflows { name } }" }),
      });
      if (res.ok) return true;
    } catch {
      /* not up yet */
    }
    await new Promise((r) => setTimeout(r, 500));
  }
  return false;
}

if (!(await waitForBackend())) {
  fail("backend did not become ready");
  stopBackend();
  process.exit(1);
}
console.log("e2e: backend ready at " + API_URL);

// 4. Run Playwright (it owns the Next.js dev server).
const pw = spawn("npx", ["playwright", "test", ...process.argv.slice(2)], {
  cwd: webDir,
  env: { ...process.env, CONDUIT_API_URL: API_URL },
  stdio: "inherit",
  shell: true,
});

pw.on("exit", (code) => {
  stopBackend();
  if (code && code !== 0) {
    fail(`playwright exited with code ${code}`);
  }
  process.exit(code ?? 1);
});
