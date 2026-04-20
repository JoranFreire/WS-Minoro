"use client";

import { useState } from "react";
import { Link } from "@/lib/api";
import { useCreateLink, useUpdateLink } from "@/hooks/useLinks";

interface LinkFormProps {
  link?: Link;
  onClose: () => void;
}

export function LinkForm({ link, onClose }: LinkFormProps) {
  const [title, setTitle] = useState(link?.title || "");
  const [fallbackUrl, setFallbackUrl] = useState(link?.fallback_url || "");
  const [strategy, setStrategy] = useState<"single" | "round_robin" | "weighted">(
    link?.routing_strategy || "round_robin"
  );
  const [isActive, setIsActive] = useState(link?.is_active ?? true);

  const createLink = useCreateLink();
  const updateLink = useUpdateLink();
  const isEditing = !!link;

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const data = {
      title,
      fallback_url: fallbackUrl,
      routing_strategy: strategy as Link["routing_strategy"],
      is_active: isActive,
    };
    if (isEditing) {
      await updateLink.mutateAsync({ id: link.id, data });
    } else {
      await createLink.mutateAsync(data);
    }
    onClose();
  };

  const isPending = createLink.isPending || updateLink.isPending;

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Title</label>
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
          placeholder="My WhatsApp Group"
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Fallback URL <span className="text-gray-400 font-normal">(optional)</span>
        </label>
        <input
          type="url"
          value={fallbackUrl}
          onChange={(e) => setFallbackUrl(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
          placeholder="https://..."
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Routing Strategy</label>
        <select
          value={strategy}
          onChange={(e) => setStrategy(e.target.value as "single" | "round_robin" | "weighted")}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
        >
          <option value="round_robin">Round Robin — distribute evenly</option>
          <option value="weighted">Weighted — distribute by weight</option>
          <option value="single">Single — always use first active</option>
        </select>
      </div>

      {isEditing && (
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium text-gray-700">Active</span>
          <button
            type="button"
            onClick={() => setIsActive(!isActive)}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              isActive ? "bg-green-600" : "bg-gray-200"
            }`}
          >
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                isActive ? "translate-x-6" : "translate-x-1"
              }`}
            />
          </button>
        </div>
      )}

      <div className="flex gap-3 pt-2">
        <button
          type="button"
          onClick={onClose}
          className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 text-sm font-medium transition-colors"
        >
          Cancel
        </button>
        <button
          type="submit"
          disabled={isPending}
          className="flex-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 text-sm font-medium transition-colors"
        >
          {isPending ? "Saving..." : isEditing ? "Update" : "Create"}
        </button>
      </div>
    </form>
  );
}
