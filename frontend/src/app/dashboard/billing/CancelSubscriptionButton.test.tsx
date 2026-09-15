import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, cleanup } from "@testing-library/react";
import { afterEach } from "vitest";
import { CancelSubscriptionButton } from "./CancelSubscriptionButton";
import * as billingHooks from "@/hooks/useBilling";

vi.mock("@/hooks/useBilling");

function mockMutation(overrides: Partial<ReturnType<typeof billingHooks.useCancelSubscription>> = {}) {
  return {
    mutate: vi.fn(),
    isPending: false,
    isSuccess: false,
    isError: false,
    ...overrides,
  } as unknown as ReturnType<typeof billingHooks.useCancelSubscription>;
}

afterEach(() => cleanup());

describe("CancelSubscriptionButton", () => {
  beforeEach(() => {
    vi.mocked(billingHooks.useCancelSubscription).mockReturnValue(mockMutation());
  });

  it("shows only a link initially, no confirmation prompt", () => {
    render(<CancelSubscriptionButton />);
    expect(screen.getByText("Cancel subscription")).toBeInTheDocument();
    expect(screen.queryByText(/are you sure/i)).not.toBeInTheDocument();
  });

  it("shows the confirmation step after clicking cancel", () => {
    render(<CancelSubscriptionButton />);
    fireEvent.click(screen.getByText("Cancel subscription"));
    expect(screen.getByText(/are you sure/i)).toBeInTheDocument();
    expect(screen.getByText("Yes, cancel")).toBeInTheDocument();
    expect(screen.getByText("Keep subscription")).toBeInTheDocument();
  });

  it("does not call mutate until the confirmation step is reached", () => {
    const mutation = mockMutation();
    vi.mocked(billingHooks.useCancelSubscription).mockReturnValue(mutation);

    render(<CancelSubscriptionButton />);
    fireEvent.click(screen.getByText("Cancel subscription"));

    expect(mutation.mutate).not.toHaveBeenCalled();
  });

  it("calls mutate only after confirming", () => {
    const mutation = mockMutation();
    vi.mocked(billingHooks.useCancelSubscription).mockReturnValue(mutation);

    render(<CancelSubscriptionButton />);
    fireEvent.click(screen.getByText("Cancel subscription"));
    fireEvent.click(screen.getByText("Yes, cancel"));

    expect(mutation.mutate).toHaveBeenCalledTimes(1);
  });

  it("dismissing the confirmation step returns to the initial state", () => {
    render(<CancelSubscriptionButton />);
    fireEvent.click(screen.getByText("Cancel subscription"));
    fireEvent.click(screen.getByText("Keep subscription"));

    expect(screen.getByText("Cancel subscription")).toBeInTheDocument();
    expect(screen.queryByText(/are you sure/i)).not.toBeInTheDocument();
  });

  it("shows an error message when the mutation failed", () => {
    vi.mocked(billingHooks.useCancelSubscription).mockReturnValue(mockMutation({ isError: true }));

    render(<CancelSubscriptionButton />);
    fireEvent.click(screen.getByText("Cancel subscription"));

    expect(screen.getByText(/could not cancel/i)).toBeInTheDocument();
  });

  it("shows a success message instead of the button once canceled", () => {
    vi.mocked(billingHooks.useCancelSubscription).mockReturnValue(mockMutation({ isSuccess: true }));

    render(<CancelSubscriptionButton />);

    expect(screen.getByText(/back on the free plan/i)).toBeInTheDocument();
    expect(screen.queryByText("Cancel subscription")).not.toBeInTheDocument();
  });
});
