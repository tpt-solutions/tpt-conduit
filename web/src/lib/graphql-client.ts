"use client";

/** Browser-side GraphQL call. Goes through /api/graphql so the httpOnly auth cookie stays server-only. */
export async function graphqlRequest<T>(
  query: string,
  variables?: Record<string, unknown>
): Promise<T> {
  const res = await fetch("/api/graphql", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ query, variables }),
  });

  if (res.status === 401) {
    window.location.href = "/login";
    throw new Error("Not authenticated");
  }

  const json = (await res.json()) as { data?: T; errors?: Array<{ message: string }> };
  if (json.errors && json.errors.length > 0) {
    throw new Error(json.errors.map((e) => e.message).join("; "));
  }
  return json.data as T;
}
