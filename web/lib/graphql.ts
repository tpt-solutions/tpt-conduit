// Minimal GraphQL client for the TPT Conduit API. Talks to the Go service's
// /graphql endpoint and attaches the single-tenant auth header.

export const API_URL =
  process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export interface GraphQLResponse<T = any> {
  data?: T;
  errors?: Array<{ message: string }>;
}

function authHeader(): Record<string, string> {
  if (typeof window === "undefined") return {};
  const key = window.localStorage.getItem("conduit_api_key");
  if (key) return { "X-API-Key": key };
  const basic = window.localStorage.getItem("conduit_basic");
  if (basic) return { Authorization: `Basic ${basic}` };
  return {};
}

export async function gql<T = any>(
  query: string,
  variables?: Record<string, any>,
): Promise<T> {
  const res = await fetch(`${API_URL}/graphql`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      ...authHeader(),
    },
    body: JSON.stringify({ query, variables }),
  });
  if (res.status === 401) {
    throw new Error("unauthorized");
  }
  const body = (await res.json()) as GraphQLResponse<T>;
  if (body.errors && body.errors.length) {
    throw new Error(body.errors.map((e) => e.message).join("; "));
  }
  return body.data as T;
}
