import { getAuthHeader } from "./auth";

const API_URL = process.env.CONDUIT_API_URL || "http://localhost:8080";

export class GraphQLAuthError extends Error {
  constructor() {
    super("Not authenticated");
    this.name = "GraphQLAuthError";
  }
}

export class GraphQLApiError extends Error {
  constructor(public errors: Array<{ message: string }>) {
    super(errors.map((e) => e.message).join("; "));
    this.name = "GraphQLApiError";
  }
}

/**
 * Server-only GraphQL client. Used directly by Server Components/Actions and
 * by the /api/graphql route handler that client components proxy through.
 */
export async function graphqlFetch<T>(
  query: string,
  variables?: Record<string, unknown>
): Promise<T> {
  const authHeader = getAuthHeader();
  if (!authHeader) {
    throw new GraphQLAuthError();
  }

  const res = await fetch(`${API_URL}/graphql`, {
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: authHeader,
    },
    body: JSON.stringify({ query, variables }),
    cache: "no-store",
  });

  if (res.status === 401) {
    throw new GraphQLAuthError();
  }

  const json = (await res.json()) as { data?: T; errors?: Array<{ message: string }> };
  if (json.errors && json.errors.length > 0) {
    throw new GraphQLApiError(json.errors);
  }
  return json.data as T;
}
