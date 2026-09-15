import { useState } from "react";
import { Destination } from "@/lib/api";
import { useAddDestination, useUpdateDestination, useDeleteDestination } from "@/hooks/useLinks";

export interface DestinationEditState {
  weight: number;
  max_clicks: string;
}

export function useDestinationList(linkId: string) {
  const [showAdd, setShowAdd] = useState(false);
  const [url, setUrl] = useState("");
  const [weight, setWeight] = useState(1);
  const [maxClicks, setMaxClicks] = useState("");
  const [editing, setEditing] = useState<Record<string, DestinationEditState>>({});

  const addDest = useAddDestination();
  const updateDest = useUpdateDestination();
  const deleteDest = useDeleteDestination();

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    await addDest.mutateAsync({
      linkId,
      data: { url, weight, max_clicks: maxClicks ? parseInt(maxClicks) : undefined },
    });
    setUrl("");
    setWeight(1);
    setMaxClicks("");
    setShowAdd(false);
  };

  const startEdit = (dest: Destination) => {
    setEditing((prev) => ({
      ...prev,
      [dest.id]: {
        weight: dest.weight,
        max_clicks: dest.max_clicks?.toString() ?? "",
      },
    }));
  };

  const cancelEdit = (id: string) => {
    setEditing((prev) => {
      const next = { ...prev };
      delete next[id];
      return next;
    });
  };

  const updateEditField = (id: string, field: keyof DestinationEditState, value: string | number) => {
    setEditing((prev) => ({
      ...prev,
      [id]: { ...prev[id], [field]: value },
    }));
  };

  const saveEdit = async (dest: Destination) => {
    const s = editing[dest.id];
    if (!s) return;
    await updateDest.mutateAsync({
      linkId,
      destId: dest.id,
      data: {
        url: dest.url,
        is_active: dest.is_active,
        weight: s.weight,
        max_clicks: s.max_clicks ? parseInt(s.max_clicks) : null,
      },
    });
    cancelEdit(dest.id);
  };

  const deleteDestination = (destId: string) => deleteDest.mutate({ linkId, destId });

  return {
    showAdd,
    setShowAdd,
    url,
    setUrl,
    weight,
    setWeight,
    maxClicks,
    setMaxClicks,
    editing,
    handleAdd,
    startEdit,
    cancelEdit,
    updateEditField,
    saveEdit,
    deleteDestination,
    isAdding: addDest.isPending,
    isSaving: updateDest.isPending,
  };
}
