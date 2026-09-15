"use client";

import { Plus } from "lucide-react";
import { Destination } from "@/lib/api";
import { useDestinationList } from "./useDestinationList";
import { AddDestinationForm } from "./AddDestinationForm";
import { DestinationRow } from "./DestinationRow";

interface DestinationListProps {
  linkId: string;
  destinations: Destination[];
  strategy: "single" | "round_robin" | "weighted";
}

export function DestinationList({ linkId, destinations, strategy }: DestinationListProps) {
  const list = useDestinationList(linkId);

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-semibold text-gray-700">
          Destinos ({destinations.length})
        </h4>
        <button
          onClick={() => list.setShowAdd(!list.showAdd)}
          className="flex items-center gap-1.5 text-xs text-green-700 hover:text-green-800 font-medium"
        >
          <Plus className="w-3.5 h-3.5" />
          Adicionar
        </button>
      </div>

      {list.showAdd && (
        <AddDestinationForm
          strategy={strategy}
          url={list.url}
          weight={list.weight}
          maxClicks={list.maxClicks}
          isPending={list.isAdding}
          onUrlChange={list.setUrl}
          onWeightChange={list.setWeight}
          onMaxClicksChange={list.setMaxClicks}
          onCancel={() => list.setShowAdd(false)}
          onSubmit={list.handleAdd}
        />
      )}

      <div className="space-y-2">
        {destinations.map((dest) => (
          <DestinationRow
            key={dest.id}
            destination={dest}
            strategy={strategy}
            editState={list.editing[dest.id]}
            isSaving={list.isSaving}
            onStartEdit={() => list.startEdit(dest)}
            onCancelEdit={() => list.cancelEdit(dest.id)}
            onSaveEdit={() => list.saveEdit(dest)}
            onEditFieldChange={(field, value) => list.updateEditField(dest.id, field, value)}
            onDelete={() => list.deleteDestination(dest.id)}
          />
        ))}

        {destinations.length === 0 && (
          <p className="text-xs text-gray-400 text-center py-4">Nenhum destino ainda</p>
        )}
      </div>
    </div>
  );
}
