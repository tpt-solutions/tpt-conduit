// Single-tenant auth: the Go API accepts either a Basic `user:pass` header or an
// `X-API-Key`. The login form captures credentials and stores the resulting
// header client-side; it is attached to every GraphQL request by lib/graphql.

export function login(username: string, password: string): void {
  if (typeof window === "undefined") return;
  const basic = btoa(`${username}:${password}`);
  window.localStorage.setItem("conduit_basic", basic);
  window.localStorage.removeItem("conduit_api_key");
}

export function loginWithKey(apiKey: string): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem("conduit_api_key", apiKey);
  window.localStorage.removeItem("conduit_basic");
}

export function logout(): void {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem("conduit_basic");
  window.localStorage.removeItem("conduit_api_key");
}

export function isAuthed(): boolean {
  if (typeof window === "undefined") return false;
  return (
    !!window.localStorage.getItem("conduit_basic") ||
    !!window.localStorage.getItem("conduit_api_key")
  );
}
