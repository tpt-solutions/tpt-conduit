import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen } from "@testing-library/react";

vi.mock("@/lib/graphql", () => ({
  graphqlFetch: vi.fn(),
}));
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push: vi.fn(), refresh: vi.fn() }),
}));

import { graphqlFetch } from "@/lib/graphql";
import NewTicketPage from "./page";
import type { Workflow } from "@/lib/types";

const mockedFetch = vi.mocked(graphqlFetch);

describe("NewTicketPage", () => {
  beforeEach(() => {
    mockedFetch.mockReset();
  });

  it("renders the ticket form populated with the fetched workflows", async () => {
    const workflows: Workflow[] = [
      { name: "it_helpdesk", version: "1.0.0", description: null, initial: "triage", steps: [] },
    ];
    mockedFetch.mockResolvedValue({ workflows });

    render(await NewTicketPage());

    expect(screen.getByRole("heading", { name: /new ticket/i })).toBeInTheDocument();
    expect(screen.getByRole("option", { name: "it_helpdesk" })).toBeInTheDocument();
  });

  it("shows the empty state when no workflows are registered", async () => {
    mockedFetch.mockResolvedValue({ workflows: [] });

    render(await NewTicketPage());
    expect(screen.getByText(/no workflows are registered/i)).toBeInTheDocument();
  });
});
