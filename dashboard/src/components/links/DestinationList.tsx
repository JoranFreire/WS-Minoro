"use client";

import { useState } from "react";
import { Destination } from "@/lib/api";
import { useAddDestination, useDeleteDestination } from "@/hooks/useLinks";
import { Plus, Trash2 } from "lucide-react";

interface DestinationListProps {
  linkId: string;
  destinations: Destination[];
}

export function DestinationList({ linkId, destinations }: DestinationListProps) {
  const [showAdd, setShowAdd] = useState(false);
  const [url, setUrl] = useState("");
  const [weight, setWeight] = useState(1);
  const [maxClicks, setMaxClicks] = useState("");

  const addDest = useAddDestination();
  const deleteDest = useDeleteDestination();

  const handleAdd = async (e: React.FormEvent) => {
    e.preventDefault();
    await addDest.mutateAsync({
      linkId,
      data: {
        url,
        weight,
        max_clicks: maxClicks ? parseInt(maxClicks) : undefined,
      },
    });
    setUrl("");
    setWeight(1);
    setMaxClicks("");
    setShowAdd(false);
  };

  return (
    <div className="space-y-3">
      <div className="flex items-center justify-between">
        <h4 className="text-sm font-semibold text-gray-700">
          Destinations ({destinations.length})
        </h4>
        <button
          onClick={() => setShowAdd(!showAdd)}
          className="flex items-center gap-1.5 text-xs text-green-700 hover:text-green-800 font-medium"
        >
          <Plus className="w-3.5 h-3.5" />
          Add
        </button>
      </div>

      {showAdd && (
        <form onSubmit={handleAdd} className="bg-gray-50 rounded-lg p-3 space-y-2 border border-gray-200">
          <input
            type="url"
            value={url}
            onChange={(e) => setUrl(e.target.value)}
            placeholder="https://chat.whatsapp.com/..."
            className="w-full px-3 py-1.5 border border-gray-300 rounded text-xs focus:outline-none focus:ring-2 focus:ring-green-500"
            required
          />
          <div className="flex gap-2">
            <div className="flex-1">
              <label className="text-xs text-gray-500 mb-0.5 block">Weight</label>
              <input
                type="number"
                value={weight}
                onChange={(e) => setWeight(parseInt(e.target.value))}
                min={1}
                className="w-full px-2 py-1.5 border border-gray-300 rounded text-xs focus:outline-none focus:ring-2 focus:ring-green-500"
              />
            </div>
            <div className="flex-1">
              <label className="text-xs text-gray-500 mb-0.5 block">Max clicks</label>
              <input
                type="number"
                value={maxClicks}
                onChange={(e) => setMaxClicks(e.target.value)}
                placeholder="∞"
                className="w-full px-2 py-1.5 border border-gray-300 rounded text-xs focus:outline-none focus:ring-2 focus:ring-green-500"
              />
            </div>
          </div>
          <div className="flex gap-2">
            <button
              type="button"
              onClick={() => setShowAdd(false)}
              className="flex-1 px-3 py-1.5 border border-gray-300 text-gray-600 rounded text-xs hover:bg-gray-100"
            >
              Cancel
            </button>
            <button
              type="submit"
              disabled={addDest.isPending}
              className="flex-1 px-3 py-1.5 bg-green-600 text-white rounded text-xs hover:bg-green-700 disabled:opacity-50"
            >
              {addDest.isPending ? "Adding..." : "Add"}
            </button>
          </div>
        </form>
      )}

      <div className="space-y-2">
        {destinations.map((dest) => (
          <div
            key={dest.id}
            className="flex items-center gap-2 bg-gray-50 rounded-lg px-3 py-2 border border-gray-100"
          >
            <div className="flex-1 min-w-0">
              <p className="text-xs font-mono text-gray-700 truncate">{dest.url}</p>
              <div className="flex gap-3 mt-0.5">
                <span className="text-xs text-gray-400">Weight: {dest.weight}</span>
                {dest.max_clicks && (
                  <span className="text-xs text-gray-400">
                    {dest.current_clicks}/{dest.max_clicks} clicks
                  </span>
                )}
              </div>
            </div>
            <button
              onClick={() => deleteDest.mutate({ linkId, destId: dest.id })}
              className="p-1 text-gray-400 hover:text-red-500 transition-colors flex-shrink-0"
            >
              <Trash2 className="w-3.5 h-3.5" />
            </button>
          </div>
        ))}
        {destinations.length === 0 && (
          <p className="text-xs text-gray-400 text-center py-4">No destinations yet</p>
        )}
      </div>
    </div>
  );
}
