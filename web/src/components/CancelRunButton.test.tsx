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
import { CancelRunButton } from "@/components/CancelRunButton";

const mockedRequest = vi.mocked(graphqlRequest);

describe("CancelRunButton", () => {
  beforeEach(() => {
    router.refresh.mockClear();
    mockedRequest.mockReset();
    mockedRequest.mockResolvedValue(undefined as never);
    vi.spyOn(window, "confirm").mockReturnValue(true);
  });

  it("does nothing when the confirm dialog is declined", async () => {
    vi.spyOn(window, "confirm").mockReturnValue(false);
    const user = userEvent.setup();
    render(<CancelRunButton runId="run-1" />);

    await user.click(screen.getByRole("button", { name: /cancel run/i }));

    expect(mockedRequest).not.toHaveBeenCalled();
    expect(router.refresh).not.toHaveBeenCalled();
  });

  it("cancels the run and refreshes when confirmed", async () => {
    const user = userEvent.setup();
    render(<CancelRunButton runId="run-1" />);

    await user.click(screen.getByRole("button", { name: /cancel run/i }));

    await waitFor(() => expect(mockedRequest).toHaveBeenCalledTimes(1));
    expect(mockedRequest).toHaveBeenCalledWith(
      expect.stringContaining("cancel("),
      expect.objectContaining({ runId: "run-1" })
    );
    expect(router.refresh).toHaveBeenCalled();
  });

  it("re-enables the button after the request settles", async () => {
    const user = userEvent.setup();
    render(<CancelRunButton runId="run-1" />);

    await user.click(screen.getByRole("button", { name: /cancel run/i }));

    await waitFor(() => expect(screen.getByRole("button")).not.toBeDisabled());
  });
});
