"use client";

import { Link } from "@/lib/api";
import { useDeleteLink } from "@/hooks/useLinks";
import { ExternalLink, Trash2, Pencil, Copy, ListChecks } from "lucide-react";

interface LinkCardProps {
  link: Link;
  onEdit: (link: Link) => void;
  onManageDestinations: (link: Link) => void;
}

const strategyLabel: Record<string, string> = {
  single: "Single",
  round_robin: "Round Robin",
  weighted: "Weighted",
};

const strategyColor: Record<string, string> = {
  single: "bg-gray-100 text-gray-700",
  round_robin: "bg-blue-50 text-blue-700",
  weighted: "bg-purple-50 text-purple-700",
};

export function LinkCard({ link, onEdit, onManageDestinations }: LinkCardProps) {
  const deleteLink = useDeleteLink();
  const routerBase = process.env.NEXT_PUBLIC_ROUTER_URL || "http://localhost:8080";
  const shortUrl = `${routerBase}/${link.short_code}`;

  const copyToClipboard = () => {
    navigator.clipboard.writeText(shortUrl);
  };

  return (
    <div className="bg-white rounded-xl border border-gray-200 p-5 hover:shadow-sm transition-shadow">
      <div className="flex items-start justify-between mb-3">
        <div className="flex-1 min-w-0">
          <h3 className="font-semibold text-gray-900 truncate">
            {link.title || link.short_code}
          </h3>
          <div className="flex items-center gap-2 mt-1 flex-wrap">
            <code className="text-xs text-green-700 bg-green-50 px-2 py-0.5 rounded font-mono">
              {link.short_code}
            </code>
            <span className={`text-xs px-2 py-0.5 rounded font-medium ${strategyColor[link.routing_strategy]}`}>
              {strategyLabel[link.routing_strategy]}
            </span>
            <span
              className={`text-xs px-2 py-0.5 rounded font-medium ${
                link.is_active ? "bg-green-50 text-green-700" : "bg-gray-100 text-gray-500"
              }`}
            >
              {link.is_active ? "Active" : "Inactive"}
            </span>
          </div>
        </div>
      </div>

      <div className="flex items-center gap-1 text-xs text-gray-500 mb-4 font-mono">
        <span className="truncate flex-1">{shortUrl}</span>
        <button onClick={copyToClipboard} className="flex-shrink-0 p-1 hover:text-gray-700">
          <Copy className="w-3 h-3" />
        </button>
      </div>

      <div className="flex items-center gap-2 pt-3 border-t border-gray-100">
        <a
          href={shortUrl}
          target="_blank"
          rel="noopener noreferrer"
          className="flex items-center gap-1.5 text-xs text-gray-600 hover:text-gray-900 transition-colors"
        >
          <ExternalLink className="w-3.5 h-3.5" />
          Test
        </a>
        <button
          onClick={() => onManageDestinations(link)}
          className="flex items-center gap-1.5 text-xs text-blue-600 hover:text-blue-800 transition-colors"
        >
          <ListChecks className="w-3.5 h-3.5" />
          Destinations
        </button>
        <div className="flex-1" />
        <button
          onClick={() => onEdit(link)}
          className="p-1.5 text-gray-400 hover:text-gray-600 rounded transition-colors"
        >
          <Pencil className="w-4 h-4" />
        </button>
        <button
          onClick={() => deleteLink.mutate(link.id)}
          className="p-1.5 text-gray-400 hover:text-red-600 rounded transition-colors"
        >
          <Trash2 className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
}
