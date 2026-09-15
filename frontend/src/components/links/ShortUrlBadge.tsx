"use client";

import { Copy } from "lucide-react";

interface ShortUrlBadgeProps {
  shortUrl: string;
}

export function ShortUrlBadge({ shortUrl }: ShortUrlBadgeProps) {
  return (
    <div className="flex items-center gap-2 bg-green-50 border border-green-200 rounded-lg px-3 py-2">
      <span className="text-xs text-green-700 font-mono flex-1 truncate">{shortUrl}</span>
      <button
        type="button"
        onClick={() => navigator.clipboard.writeText(shortUrl)}
        className="text-green-600 hover:text-green-800 flex-shrink-0"
        title="Copiar link"
      >
        <Copy className="w-3.5 h-3.5" />
      </button>
    </div>
  );
}
