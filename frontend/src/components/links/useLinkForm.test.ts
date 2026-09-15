import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useLinkForm } from "./useLinkForm";
import * as useLinksModule from "@/hooks/useLinks";
import type { Link } from "@/lib/api";

vi.mock("@/hooks/useLinks");

function mockMutation(overrides: Partial<{ mutateAsync: ReturnType<typeof vi.fn>; isPending: boolean; error: unknown }> = {}) {
  return {
    mutateAsync: vi.fn().mockResolvedValue({ id: "new-link-id" }),
    isPending: false,
    error: null,
    ...overrides,
  };
}

function asMutation<T>(mock: ReturnType<typeof mockMutation>): T {
  return mock as unknown as T;
}

describe("useLinkForm", () => {
  let createLink: ReturnType<typeof mockMutation>;
  let updateLink: ReturnType<typeof mockMutation>;
  let addDestination: ReturnType<typeof mockMutation>;

  beforeEach(() => {
    createLink = mockMutation();
    updateLink = mockMutation();
    addDestination = mockMutation();

    vi.mocked(useLinksModule.useCreateLink).mockReturnValue(
      asMutation<ReturnType<typeof useLinksModule.useCreateLink>>(createLink)
    );
    vi.mocked(useLinksModule.useUpdateLink).mockReturnValue(
      asMutation<ReturnType<typeof useLinksModule.useUpdateLink>>(updateLink)
    );
    vi.mocked(useLinksModule.useAddDestination).mockReturnValue(
      asMutation<ReturnType<typeof useLinksModule.useAddDestination>>(addDestination)
    );
  });

  it("defaults to create-mode state when no link is passed", () => {
    const onClose = vi.fn();
    const { result } = renderHook(() => useLinkForm({ onClose }));

    expect(result.current.isEditing).toBe(false);
    expect(result.current.title).toBe("");
    expect(result.current.strategy).toBe("round_robin");
    expect(result.current.isActive).toBe(true);
    expect(result.current.destinations).toEqual([{ url: "", weight: 1, max_clicks: "" }]);
  });

  it("pre-fills state from the given link in edit mode", () => {
    const link: Link = {
      id: "1",
      tenant_id: "t1",
      short_code: "abc123",
      title: "Existing",
      fallback_url: "https://fallback.example.com",
      routing_strategy: "weighted",
      is_active: false,
      created_at: "",
      updated_at: "",
    };
    const { result } = renderHook(() => useLinkForm({ link, onClose: vi.fn() }));

    expect(result.current.isEditing).toBe(true);
    expect(result.current.title).toBe("Existing");
    expect(result.current.fallbackUrl).toBe("https://fallback.example.com");
    expect(result.current.strategy).toBe("weighted");
    expect(result.current.isActive).toBe(false);
  });

  it("addRow appends a blank destination draft", () => {
    const { result } = renderHook(() => useLinkForm({ onClose: vi.fn() }));

    act(() => result.current.addRow());

    expect(result.current.destinations).toHaveLength(2);
  });

  it("removeRow removes the destination at the given index", () => {
    const { result } = renderHook(() => useLinkForm({ onClose: vi.fn() }));
    act(() => result.current.addRow());
    act(() => result.current.updateRow(0, "url", "https://first.example.com"));
    act(() => result.current.updateRow(1, "url", "https://second.example.com"));

    act(() => result.current.removeRow(0));

    expect(result.current.destinations).toEqual([
      { url: "https://second.example.com", weight: 1, max_clicks: "" },
    ]);
  });

  it("updateRow updates only the targeted field of the targeted row", () => {
    const { result } = renderHook(() => useLinkForm({ onClose: vi.fn() }));

    act(() => result.current.updateRow(0, "url", "https://example.com"));
    act(() => result.current.updateRow(0, "weight", 5));

    expect(result.current.destinations[0]).toEqual({
      url: "https://example.com",
      weight: 5,
      max_clicks: "",
    });
  });

  it("create mode: submits the link then only the destinations with a non-empty url", async () => {
    const onClose = vi.fn();
    const { result } = renderHook(() => useLinkForm({ onClose }));

    act(() => result.current.updateRow(0, "url", "https://first.example.com"));
    act(() => result.current.addRow());
    // second row left blank on purpose — must be filtered out

    await act(async () => {
      await result.current.handleSubmit({ preventDefault: vi.fn() } as unknown as React.FormEvent);
    });

    expect(createLink.mutateAsync).toHaveBeenCalledTimes(1);
    expect(addDestination.mutateAsync).toHaveBeenCalledTimes(1);
    expect(addDestination.mutateAsync).toHaveBeenCalledWith({
      linkId: "new-link-id",
      data: { url: "https://first.example.com", weight: 1, max_clicks: undefined },
    });
    expect(updateLink.mutateAsync).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("edit mode: submits an update and never touches destination endpoints", async () => {
    const link: Link = {
      id: "link-1",
      tenant_id: "t1",
      short_code: "abc123",
      title: "Existing",
      fallback_url: "",
      routing_strategy: "single",
      is_active: true,
      created_at: "",
      updated_at: "",
    };
    const onClose = vi.fn();
    const { result } = renderHook(() => useLinkForm({ link, onClose }));

    await act(async () => {
      await result.current.handleSubmit({ preventDefault: vi.fn() } as unknown as React.FormEvent);
    });

    expect(updateLink.mutateAsync).toHaveBeenCalledWith({
      id: "link-1",
      data: {
        title: "Existing",
        fallback_url: "",
        routing_strategy: "single",
        is_active: true,
      },
    });
    expect(createLink.mutateAsync).not.toHaveBeenCalled();
    expect(addDestination.mutateAsync).not.toHaveBeenCalled();
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("isPending is true when any of the three mutations is pending", () => {
    vi.mocked(useLinksModule.useAddDestination).mockReturnValue(
      asMutation<ReturnType<typeof useLinksModule.useAddDestination>>(mockMutation({ isPending: true }))
    );
    const { result } = renderHook(() => useLinkForm({ onClose: vi.fn() }));

    expect(result.current.isPending).toBe(true);
  });
});
