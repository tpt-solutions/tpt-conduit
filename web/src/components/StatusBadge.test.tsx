import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { StatusBadge } from "@/components/StatusBadge";

describe("StatusBadge", () => {
  it("renders the status text", () => {
    render(<StatusBadge status="RUNNING" />);
    const badge = screen.getByText("RUNNING");
    expect(badge).toBeInTheDocument();
    expect(badge).toHaveClass("badge");
  });

  it("maps a running status to the active tone", () => {
    render(<StatusBadge status="running" />);
    expect(screen.getByText("running")).toHaveClass("badge-active");
  });

  it("maps a completed status to the success tone", () => {
    render(<StatusBadge status="COMPLETED" />);
    expect(screen.getByText("COMPLETED")).toHaveClass("badge-success");
  });

  it("maps a failed status to the danger tone", () => {
    render(<StatusBadge status="FAILED" />);
    expect(screen.getByText("FAILED")).toHaveClass("badge-danger");
  });

  it("maps a waiting status to the warning tone", () => {
    render(<StatusBadge status="WAITING" />);
    expect(screen.getByText("WAITING")).toHaveClass("badge-warning");
  });

  it("falls back to a neutral tone for unknown statuses", () => {
    render(<StatusBadge status="MYSTERY" />);
    expect(screen.getByText("MYSTERY")).toHaveClass("badge-neutral");
  });
});
