"use client";

import { Plus, Trash2 } from "lucide-react";
import { Link } from "@/lib/api";
import { DestinationDraft } from "./useLinkForm";
import { INPUT, INPUT_SM } from "./formStyles";

interface DestinationDraftEditorProps {
  strategy: Link["routing_strategy"];
  destinations: DestinationDraft[];
  onAdd: () => void;
  onRemove: (index: number) => void;
  onUpdate: (index: number, field: keyof DestinationDraft, value: string | number) => void;
}

export function DestinationDraftEditor({
  strategy,
  destinations,
  onAdd,
  onRemove,
  onUpdate,
}: DestinationDraftEditorProps) {
  return (
    <div>
      <div className="flex items-center justify-between mb-2">
        <label className="block text-sm font-medium text-gray-700">Links de destino</label>
        <button
          type="button"
          onClick={onAdd}
          className="flex items-center gap-1 text-xs text-green-700 hover:text-green-800 font-medium"
        >
          <Plus className="w-3.5 h-3.5" />
          Adicionar
        </button>
      </div>
      <div className="space-y-2">
        {destinations.map((dest, i) => (
          <div key={i} className="flex gap-2 items-start">
            <div className="flex-1 space-y-1">
              <input
                type="url"
                value={dest.url}
                onChange={(e) => onUpdate(i, "url", e.target.value)}
                placeholder="https://chat.whatsapp.com/..."
                className={INPUT}
              />
              {strategy !== "single" && (
                <div className="flex gap-2">
                  {strategy === "weighted" && (
                    <div className="flex-1">
                      <input
                        type="number"
                        value={dest.weight}
                        onChange={(e) => onUpdate(i, "weight", parseInt(e.target.value))}
                        min={1}
                        placeholder="Peso"
                        className={INPUT_SM}
                      />
                    </div>
                  )}
                  <div className="flex-1">
                    <input
                      type="number"
                      value={dest.max_clicks}
                      onChange={(e) => onUpdate(i, "max_clicks", e.target.value)}
                      placeholder="Máx. cliques (opcional)"
                      className={INPUT_SM}
                    />
                  </div>
                </div>
              )}
            </div>
            {destinations.length > 1 && (
              <button
                type="button"
                onClick={() => onRemove(i)}
                className="mt-2 p-1 text-gray-400 hover:text-red-500"
              >
                <Trash2 className="w-4 h-4" />
              </button>
            )}
          </div>
        ))}
      </div>
    </div>
  );
}
