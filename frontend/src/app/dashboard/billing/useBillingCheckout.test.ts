import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useBillingCheckout } from "./useBillingCheckout";
import * as apiModule from "@/lib/api";

const push = vi.fn();
vi.mock("next/navigation", () => ({
  useRouter: () => ({ push }),
}));

vi.mock("@/lib/api", () => ({
  subscribeToPlan: vi.fn(),
}));

function getRegisteredCallbacks() {
  const initMock = window.PagarmeCheckout!.init as ReturnType<typeof vi.fn>;
  const [success, fail] = initMock.mock.calls[0];
  return { success, fail };
}

describe("useBillingCheckout", () => {
  beforeEach(() => {
    push.mockClear();
    vi.mocked(apiModule.subscribeToPlan).mockReset();
    window.PagarmeCheckout = { init: vi.fn() };
  });

  it("defaults to the starter plan", () => {
    const { result } = renderHook(() => useBillingCheckout());
    expect(result.current.plan).toBe("starter");
  });

  it("registers PagarmeCheckout.init exactly once when the script loads", () => {
    const { result } = renderHook(() => useBillingCheckout());

    act(() => result.current.onScriptLoad());
    act(() => result.current.onScriptLoad());

    expect(window.PagarmeCheckout!.init).toHaveBeenCalledTimes(1);
  });

  it("submits the currently selected plan, not whatever it was when the script loaded", async () => {
    vi.mocked(apiModule.subscribeToPlan).mockResolvedValue({
      data: { plan: "pro", quota_clicks_month: 500_000 },
    } as never);

    const { result } = renderHook(() => useBillingCheckout());
    act(() => result.current.onScriptLoad());

    // User changes the plan *after* the script (and its callback) loaded.
    act(() => result.current.setPlan("pro"));

    const { success } = getRegisteredCallbacks();
    await act(async () => {
      success({ id: "token_abc123" });
    });

    expect(apiModule.subscribeToPlan).toHaveBeenCalledWith("pro", "token_abc123");
  });

  it("extracts the token from a nested token.id shape", async () => {
    vi.mocked(apiModule.subscribeToPlan).mockResolvedValue({ data: {} } as never);
    const { result } = renderHook(() => useBillingCheckout());
    act(() => result.current.onScriptLoad());

    const { success } = getRegisteredCallbacks();
    await act(async () => {
      success({ token: { id: "token_nested" } });
    });

    expect(apiModule.subscribeToPlan).toHaveBeenCalledWith("starter", "token_nested");
  });

  it("shows an error and never calls the API when no token can be found", async () => {
    const { result } = renderHook(() => useBillingCheckout());
    act(() => result.current.onScriptLoad());

    const { success } = getRegisteredCallbacks();
    act(() => {
      success({});
    });

    expect(apiModule.subscribeToPlan).not.toHaveBeenCalled();
    expect(result.current.error).not.toBe("");
  });

  it("redirects to settings after a successful subscription", async () => {
    vi.mocked(apiModule.subscribeToPlan).mockResolvedValue({ data: {} } as never);
    const { result } = renderHook(() => useBillingCheckout());
    act(() => result.current.onScriptLoad());

    const { success } = getRegisteredCallbacks();
    await act(async () => {
      success({ id: "token_abc" });
    });

    expect(push).toHaveBeenCalledWith("/dashboard/settings");
  });

  it("shows the API's error message on failure and does not redirect", async () => {
    vi.mocked(apiModule.subscribeToPlan).mockRejectedValue({
      response: { data: { error: "card declined" } },
    });
    const { result } = renderHook(() => useBillingCheckout());
    act(() => result.current.onScriptLoad());

    const { success } = getRegisteredCallbacks();
    await act(async () => {
      success({ id: "token_abc" });
    });

    expect(result.current.error).toBe("card declined");
    expect(push).not.toHaveBeenCalled();
  });
});
