import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { useDestinationList } from "./useDestinationList";
import * as useLinksModule from "@/hooks/useLinks";
import type { Destination } from "@/lib/api";

vi.mock("@/hooks/useLinks");

function mockMutation(overrides: Partial<{ mutateAsync: ReturnType<typeof vi.fn>; mutate: ReturnType<typeof vi.fn>; isPending: boolean }> = {}) {
  return {
    mutateAsync: vi.fn().mockResolvedValue(undefined),
    mutate: vi.fn(),
    isPending: false,
    ...overrides,
  };
}

function asMutation<T>(mock: ReturnType<typeof mockMutation>): T {
  return mock as unknown as T;
}

function makeDestination(overrides: Partial<Destination> = {}): Destination {
  return {
    id: "dest-1",
    link_id: "link-1",
    url: "https://example.com",
    weight: 1,
    max_clicks: null,
    current_clicks: 0,
    is_active: true,
    ...overrides,
  };
}

describe("useDestinationList", () => {
  let addDest: ReturnType<typeof mockMutation>;
  let updateDest: ReturnType<typeof mockMutation>;
  let deleteDest: ReturnType<typeof mockMutation>;

  beforeEach(() => {
    addDest = mockMutation();
    updateDest = mockMutation();
    deleteDest = mockMutation();

    vi.mocked(useLinksModule.useAddDestination).mockReturnValue(
      asMutation<ReturnType<typeof useLinksModule.useAddDestination>>(addDest)
    );
    vi.mocked(useLinksModule.useUpdateDestination).mockReturnValue(
      asMutation<ReturnType<typeof useLinksModule.useUpdateDestination>>(updateDest)
    );
    vi.mocked(useLinksModule.useDeleteDestination).mockReturnValue(
      asMutation<ReturnType<typeof useLinksModule.useDeleteDestination>>(deleteDest)
    );
  });

  it("handleAdd submits the draft and resets the form", async () => {
    const { result } = renderHook(() => useDestinationList("link-1"));

    act(() => result.current.setUrl("https://new.example.com"));
    act(() => result.current.setWeight(3));
    act(() => result.current.setMaxClicks("50"));
    act(() => result.current.setShowAdd(true));

    await act(async () => {
      await result.current.handleAdd({ preventDefault: vi.fn() } as unknown as React.FormEvent);
    });

    expect(addDest.mutateAsync).toHaveBeenCalledWith({
      linkId: "link-1",
      data: { url: "https://new.example.com", weight: 3, max_clicks: 50 },
    });
    expect(result.current.url).toBe("");
    expect(result.current.weight).toBe(1);
    expect(result.current.maxClicks).toBe("");
    expect(result.current.showAdd).toBe(false);
  });

  it("startEdit seeds the edit state from the destination", () => {
    const { result } = renderHook(() => useDestinationList("link-1"));
    const dest = makeDestination({ id: "d1", weight: 4, max_clicks: 200 });

    act(() => result.current.startEdit(dest));

    expect(result.current.editing["d1"]).toEqual({ weight: 4, max_clicks: "200" });
  });

  it("startEdit handles a destination with no max_clicks", () => {
    const { result } = renderHook(() => useDestinationList("link-1"));
    const dest = makeDestination({ id: "d1", max_clicks: null });

    act(() => result.current.startEdit(dest));

    expect(result.current.editing["d1"].max_clicks).toBe("");
  });

  it("cancelEdit removes the destination from the editing map", () => {
    const { result } = renderHook(() => useDestinationList("link-1"));
    const dest = makeDestination({ id: "d1" });
    act(() => result.current.startEdit(dest));

    act(() => result.current.cancelEdit("d1"));

    expect(result.current.editing["d1"]).toBeUndefined();
  });

  it("updateEditField updates only the given field for the given destination", () => {
    const { result } = renderHook(() => useDestinationList("link-1"));
    act(() => result.current.startEdit(makeDestination({ id: "d1", weight: 1 })));

    act(() => result.current.updateEditField("d1", "weight", 9));

    expect(result.current.editing["d1"].weight).toBe(9);
  });

  it("saveEdit submits the edited fields and clears the editing state", async () => {
    const { result } = renderHook(() => useDestinationList("link-1"));
    const dest = makeDestination({ id: "d1", url: "https://x.example.com", is_active: true });
    act(() => result.current.startEdit(dest));
    act(() => result.current.updateEditField("d1", "weight", 7));

    await act(async () => {
      await result.current.saveEdit(dest);
    });

    expect(updateDest.mutateAsync).toHaveBeenCalledWith({
      linkId: "link-1",
      destId: "d1",
      data: { url: "https://x.example.com", is_active: true, weight: 7, max_clicks: null },
    });
    expect(result.current.editing["d1"]).toBeUndefined();
  });

  it("saveEdit is a no-op when there is no pending edit for that destination", async () => {
    const { result } = renderHook(() => useDestinationList("link-1"));
    const dest = makeDestination({ id: "d1" });

    await act(async () => {
      await result.current.saveEdit(dest);
    });

    expect(updateDest.mutateAsync).not.toHaveBeenCalled();
  });

  it("deleteDestination delegates to the delete mutation with linkId and destId", () => {
    const { result } = renderHook(() => useDestinationList("link-1"));

    act(() => result.current.deleteDestination("d1"));

    expect(deleteDest.mutate).toHaveBeenCalledWith({ linkId: "link-1", destId: "d1" });
  });
});
