"use client";

import { useState } from "react";
import { Destination } from "@/lib/api";
import { useAddDestination, useUpdateDestination, useDeleteDestination } from "@/hooks/useLinks";
import { Plus, Trash2, Pencil, Check, X } from "lucide-react";

interface DestinationListProps {
  linkId: string;
  destinations: Destination[];
  strategy: "single" | "round_robin" | "weighted";
}

const INPUT_SM = "w-full px-2 py-1.5 border border-gray-300 rounded text-xs text-gray-900 bg-white focus:outline-none focus:ring-2 focus:ring-green-500";

interface EditState {
  weight: number;
  max_clicks: string;
}

export function DestinationList({ linkId, destinations, strategy }: DestinationListProps) {
  const [showAdd, setShowAdd] = useState(false);
  const [url, setUrl] = useState("");
  const [weight, setWeight] = useState(1);
  const [maxClicks, setMaxClicks] = useState("");
  const [editing, setEditing] = useState<Record<string, EditState>>({});

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

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-semibold text-gray-700">
          Destinos ({destinations.length})
        </h4>
        <button
          onClick={() => setShowAdd(!showAdd)}
          className="flex items-center gap-1.5 text-xs text-green-700 hover:text-green-800 font-medium"
        >
          <Plus className="w-3.5 h-3.5" />
          Adicionar
        </button>
      </div>

      {showAdd && (
        <form onSubmit={handleAdd} className="bg-gray-50 rounded-lg p-3 space-y-2 border border-gray-200">
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://chat.whatsapp.com/..."
            className={INPUT_SM}
            required
          />
          <div className="flex gap-2">
            {strategy === "weighted" && (
              <div className="flex-1">
                <label className="text-xs text-gray-500 mb-0.5 block">Peso</label>
                <input
                  type="number"
                  value={weight}
                  onChange={(e) => setWeight(parseInt(e.target.value))}
                  min={1}
                  className={INPUT_SM}
                />
              </div>
            )}
            <div className="flex-1">
              <label className="text-xs text-gray-500 mb-0.5 block">Máx. cliques</label>
              <input
                type="number"
                value={maxClicks}
                onChange={(e) => setMaxClicks(e.target.value)}
                placeholder="∞"
                className={INPUT_SM}
              />
            </div>
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setShowAdd(false)}
              className="flex-1 px-3 py-1.5 border border-gray-300 text-gray-600 rounded text-xs hover:bg-gray-100"
            >
              Cancelar
            </button>
            <button
              type="submit"
              disabled={addDest.isPending}
              className="flex-1 px-3 py-1.5 bg-green-600 text-white rounded text-xs hover:bg-green-700 disabled:opacity-50"
            >
              {addDest.isPending ? "Adicionando..." : "Adicionar"}
            </button>
          </div>
        </form>
      )}

      <div className="space-y-2">
        {destinations.map((dest) => {
          const isEditing = !!editing[dest.id];
          const editState = editing[dest.id];

          return (
            <div
              key={dest.id}
              className="bg-gray-50 rounded-lg px-3 py-2 border border-gray-100"
            >
              <div className="flex items-start gap-2">
                <p className="text-xs font-mono text-gray-700 truncate flex-1">{dest.url}</p>
                <div className="flex gap-1 flex-shrink-0">
                  {isEditing ? (
                    <>
                      <button
                        onClick={() => saveEdit(dest)}
                        disabled={updateDest.isPending}
                        className="p-1 text-green-600 hover:text-green-700"
                      >
                        <Check className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => cancelEdit(dest.id)}
                        className="p-1 text-gray-400 hover:text-gray-600"
                      >
                        <X className="w-3.5 h-3.5" />
                      </button>
                    </>
                  ) : (
                    <>
                      <button
                        onClick={() => startEdit(dest)}
                        className="p-1 text-gray-400 hover:text-gray-600"
                      >
                        <Pencil className="w-3.5 h-3.5" />
                      </button>
                      <button
                        onClick={() => deleteDest.mutate({ linkId, destId: dest.id })}
                        className="p-1 text-gray-400 hover:text-red-500"
                      >
                        <Trash2 className="w-3.5 h-3.5" />
                      </button>
                    </>
                  )}
                </div>
              </div>

              {isEditing ? (
                <div className="flex gap-2 mt-2">
                  {strategy === "weighted" && (
                    <div className="flex-1">
                      <label className="text-xs text-gray-500 mb-0.5 block">Peso</label>
                      <input
                        type="number"
                        value={editState.weight}
                        onChange={(e) =>
                          setEditing((prev) => ({
                            ...prev,
                            [dest.id]: { ...prev[dest.id], weight: parseInt(e.target.value) },
                          }))
                        }
                        min={1}
                        className={INPUT_SM}
                      />
                    </div>
                  )}
                  <div className="flex-1">
                    <label className="text-xs text-gray-500 mb-0.5 block">Máx. cliques</label>
                    <input
                      type="number"
                      value={editState.max_clicks}
                      onChange={(e) =>
                        setEditing((prev) => ({
                          ...prev,
                          [dest.id]: { ...prev[dest.id], max_clicks: e.target.value },
                        }))
                      }
                      placeholder="∞"
                      className={INPUT_SM}
                    />
                  </div>
                </div>
              ) : (
                <div className="flex gap-3 mt-0.5">
                  {strategy === "weighted" && (
                    <span className="text-xs text-gray-400">Peso: {dest.weight}</span>
                  )}
                  <span className="text-xs text-gray-400">
                    {dest.max_clicks
                      ? `${dest.current_clicks}/${dest.max_clicks} cliques`
                      : `${dest.current_clicks} cliques`}
                  </span>
                </div>
              )}
            </div>
          );
        })}

        {destinations.length === 0 && (
          <p className="text-xs text-gray-400 text-center py-4">Nenhum destino ainda</p>
        )}
      </div>
    </div>
  );
}
