"use client";

import { Trash2, Pencil, Check, X } from "lucide-react";
import { Destination, Link } from "@/lib/api";
import { DestinationEditState } from "./useDestinationList";
import { INPUT_SM } from "./formStyles";

interface DestinationRowProps {
  destination: Destination;
  strategy: Link["routing_strategy"];
  editState?: DestinationEditState;
  isSaving: boolean;
  onStartEdit: () => void;
  onCancelEdit: () => void;
  onSaveEdit: () => void;
  onEditFieldChange: (field: keyof DestinationEditState, value: string | number) => void;
  onDelete: () => void;
}

export function DestinationRow({
  destination,
  strategy,
  editState,
  isSaving,
  onStartEdit,
  onCancelEdit,
  onSaveEdit,
  onEditFieldChange,
  onDelete,
}: DestinationRowProps) {
  const isEditing = !!editState;

  return (
    <div className="bg-gray-50 rounded-lg px-3 py-2 border border-gray-100">
      <div className="flex items-start gap-2">
        <p className="text-xs font-mono text-gray-700 truncate flex-1">{destination.url}</p>
        <div className="flex gap-1 flex-shrink-0">
          {isEditing ? (
            <>
              <button onClick={onSaveEdit} disabled={isSaving} className="p-1 text-green-600 hover:text-green-700">
                <Check className="w-3.5 h-3.5" />
              </button>
              <button onClick={onCancelEdit} className="p-1 text-gray-400 hover:text-gray-600">
                <X className="w-3.5 h-3.5" />
              </button>
            </>
          ) : (
            <>
              <button onClick={onStartEdit} className="p-1 text-gray-400 hover:text-gray-600">
                <Pencil className="w-3.5 h-3.5" />
              </button>
              <button onClick={onDelete} className="p-1 text-gray-400 hover:text-red-500">
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
                onChange={(e) => onEditFieldChange("weight", parseInt(e.target.value))}
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
              onChange={(e) => onEditFieldChange("max_clicks", e.target.value)}
              placeholder="∞"
              className={INPUT_SM}
            />
          </div>
        </div>
      ) : (
        <div className="flex gap-3 mt-0.5">
          {strategy === "weighted" && (
            <span className="text-xs text-gray-400">Peso: {destination.weight}</span>
          )}
          <span className="text-xs text-gray-400">
            {destination.max_clicks
              ? `${destination.current_clicks}/${destination.max_clicks} cliques`
              : `${destination.current_clicks} cliques`}
          </span>
        </div>
      )}
    </div>
  );
}
