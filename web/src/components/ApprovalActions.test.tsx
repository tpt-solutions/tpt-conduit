import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

const router = { push: vi.fn(), refresh: vi.fn() };
vi.mock("next/navigation", () => ({
  useRouter: () => router,
}));
vi.mock("@/lib/graphql-client", () => ({
  graphqlRequest: vi.fn(),
}));

import { graphqlRequest } from "@/lib/graphql-client";
import { ApprovalActions } from "@/components/ApprovalActions";

const mockedRequest = vi.mocked(graphqlRequest);

describe("ApprovalActions", () => {
  beforeEach(() => {
    router.push.mockClear();
    router.refresh.mockClear();
    mockedRequest.mockReset();
    mockedRequest.mockResolvedValue(undefined as never);
  });

  it("shows an error and does not call the API when 'by' is empty", async () => {
    const user = userEvent.setup();
    render(<ApprovalActions runId="run-1" step="manager_approval" />);

    await user.click(screen.getByRole("button", { name: /approve/i }));

    expect(await screen.findByText(/enter who is deciding/i)).toBeInTheDocument();
    expect(mockedRequest).not.toHaveBeenCalled();
  });

  it("calls approve with the expected variables and refreshes", async () => {
    const user = userEvent.setup();

    render(<ApprovalActions runId="run-1" step="manager_approval" />);

    await user.type(screen.getByLabelText(/decided by/i), "boss@example.com");
    await user.click(screen.getByRole("button", { name: /approve/i }));

    await waitFor(() => expect(mockedRequest).toHaveBeenCalledTimes(1));
    expect(mockedRequest).toHaveBeenCalledWith(
      expect.stringContaining("approve("),
      expect.objectContaining({ runId: "run-1", step: "manager_approval", by: "boss@example.com" })
    );
    expect(router.refresh).toHaveBeenCalled();
  });

  it("calls reject with a reason and refreshes", async () => {
    const user = userEvent.setup();

    render(<ApprovalActions runId="run-1" step="manager_approval" />);

    await user.type(screen.getByLabelText(/decided by/i), "boss@example.com");
    await user.type(screen.getByLabelText(/comment \/ reason/i), "nope");
    await user.click(screen.getByRole("button", { name: /reject/i }));

    await waitFor(() => expect(mockedRequest).toHaveBeenCalledTimes(1));
    expect(mockedRequest).toHaveBeenCalledWith(
      expect.stringContaining("reject("),
      expect.objectContaining({
        runId: "run-1",
        step: "manager_approval",
        by: "boss@example.com",
        reason: "nope",
      })
    );
    expect(router.refresh).toHaveBeenCalled();
  });

  it("surfaces API errors", async () => {
    const user = userEvent.setup();
    mockedRequest.mockRejectedValue(new Error("not authorized"));

    render(<ApprovalActions runId="run-1" step="manager_approval" />);

    await user.type(screen.getByLabelText(/decided by/i), "someone");
    await user.click(screen.getByRole("button", { name: /approve/i }));

    expect(await screen.findByText(/not authorized/i)).toBeInTheDocument();
    expect(mockedRequest).toHaveBeenCalledTimes(1);
  });
});
