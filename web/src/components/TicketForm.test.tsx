import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const router = { push: vi.fn(), refresh: vi.fn() };
vi.mock("next/navigation", () => ({
  useRouter: () => router,
}));
vi.mock("@/lib/graphql-client", () => ({
  graphqlRequest: vi.fn(),
}));

import { graphqlRequest } from "@/lib/graphql-client";
import { TicketForm } from "@/components/TicketForm";
import type { Workflow } from "@/lib/types";

const mockedRequest = vi.mocked(graphqlRequest);

const workflows: Workflow[] = [
  {
    name: "it_helpdesk",
    version: "1.0.0",
    description: null,
    initial: "triage",
    steps: [],
  },
  {
    name: "it_helpdesk",
    version: "1.1.0",
    description: null,
    initial: "triage",
    steps: [],
  },
  {
    name: "hr_onboarding",
    version: "2.0.0",
    description: null,
    initial: "start",
    steps: [],
  },
];

describe("TicketForm", () => {
  beforeEach(() => {
    router.push.mockClear();
    router.refresh.mockClear();
    mockedRequest.mockReset();
    mockedRequest.mockResolvedValue({ createTicket: { id: "ticket-123" } } as never);
  });

  it("shows an empty state when no workflows are registered", () => {
    render(<TicketForm workflows={[]} />);
    expect(screen.getByText(/no workflows are registered/i)).toBeInTheDocument();
  });

  it("groups workflows by name and exposes their versions", async () => {
    const user = userEvent.setup();
    render(<TicketForm workflows={workflows} />);
    const options = () => screen.getAllByRole("option").map((o) => o.textContent);
    expect(options()).toContain("it_helpdesk");
    expect(options()).toContain("hr_onboarding");

    // The version dropdown only lists versions for the selected workflow.
    expect(options()).toContain("1.0.0");
    expect(options()).toContain("1.1.0");
    expect(options()).not.toContain("2.0.0");

    await user.selectOptions(screen.getByLabelText(/workflow/i), "hr_onboarding");
    expect(options()).toContain("2.0.0");
    expect(options()).not.toContain("1.0.0");
  });

  it("rejects invalid JSON in the fields field", async () => {
    const user = userEvent.setup();
    render(<TicketForm workflows={workflows} />);

    await user.type(screen.getByLabelText(/title/i), "Broken laptop");
    fireEvent.change(screen.getByLabelText(/fields/i), { target: { value: "{ not valid json" } });
    await user.click(screen.getByRole("button", { name: /create ticket/i }));

    expect(await screen.findByText(/fields must be valid json/i)).toBeInTheDocument();
    expect(mockedRequest).not.toHaveBeenCalled();
  });

  it("creates a ticket and navigates on a valid submit", async () => {
    const user = userEvent.setup();

    render(<TicketForm workflows={workflows} />);

    await user.type(screen.getByLabelText(/title/i), "Broken laptop");
    fireEvent.change(screen.getByLabelText(/fields/i), { target: { value: '{"urgency":"high"}' } });
    await user.click(screen.getByRole("button", { name: /create ticket/i }));

    await waitFor(() => expect(mockedRequest).toHaveBeenCalledTimes(1));
    expect(mockedRequest).toHaveBeenCalledWith(
      expect.stringContaining("createTicket("),
      expect.objectContaining({
        input: expect.objectContaining({
          workflow: "it_helpdesk",
          version: "1.0.0",
          title: "Broken laptop",
          fields: { urgency: "high" },
        }),
      })
    );
    expect(router.push).toHaveBeenCalledWith("/tickets/ticket-123");
  });
});
