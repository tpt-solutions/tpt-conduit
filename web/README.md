# /web — Frontend (Phase 4)

Next.js (App Router, TypeScript) console for TPT Conduit: ticket list/detail/
create, workflow-run visualization with an approval/reject/cancel UI,
white-label theming, and a login screen backed by the Go API's Basic-auth /
API-key auth.

## Setup

```
cd web
npm install
cp .env.example .env.local   # point CONDUIT_API_URL at the running Go API
npm run dev
```

The Go API has no login/token endpoint (see `api/auth.go`) — it checks a
static username/password or API key on every request. The Next.js server
verifies the submitted credential against the backend once at `/login`, then
stores the resulting `Authorization` header value in an httpOnly cookie.
Client components never see the credential directly; they call the same-origin
`/api/graphql` route, which attaches the header server-side before forwarding
to `CONDUIT_API_URL/graphql`.

## Layout

- `src/app/login` — login page (public).
- `src/app/(site)` — authenticated pages (tickets, runs), behind `middleware.ts`.
- `src/app/api/{login,logout,graphql}` — route handlers for auth + the GraphQL proxy.
- `src/lib/graphql.ts` — server-side GraphQL client (Server Components).
- `src/lib/graphql-client.ts` — browser-side client (Client Components, via the proxy).
- `src/lib/queries.ts` — all GraphQL query/mutation strings, matched to the schema in `api/graphql`.
- `src/lib/theme.ts` — white-label config (brand name/logo/colors), read from `NEXT_PUBLIC_*` env vars.
- `src/lib/plugins/` — placeholder plugin API surface only; nothing loads it yet (see its README).

## Known gaps vs. the backend

- No ticket **update** mutation exists on the backend yet (`createTicket` only) — the ticket detail page is read-only.
- No pagination/filtering on `tickets`/`runs` queries — lists fetch everything the backend returns.
