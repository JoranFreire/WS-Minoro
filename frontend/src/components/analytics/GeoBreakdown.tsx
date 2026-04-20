"use client";

import { ClickByCountry } from "@/hooks/useAnalytics";

interface GeoBreakdownProps {
  data: ClickByCountry[];
}

export function GeoBreakdown({ data }: GeoBreakdownProps) {
  const total = data.reduce((sum, row) => sum + row.clicks, 0);

  if (data.length === 0) {
    return <p className="text-sm text-gray-400 py-4 text-center">No country data yet</p>;
  }

  return (
    <div className="space-y-2">
      {data.map((row) => {
        const pct = total > 0 ? Math.round((row.clicks / total) * 100) : 0;
        return (
          <div key={row.country_code}>
            <div className="flex justify-between text-sm mb-1">
              <span className="font-medium text-gray-700">{row.country_code}</span>
              <span className="text-gray-500">
                {row.clicks.toLocaleString()} ({pct}%)
              </span>
            </div>
            <div className="h-1.5 bg-gray-100 rounded-full overflow-hidden">
              <div
                className="h-full bg-green-500 rounded-full"
                style={{ width: `${pct}%` }}
              />
            </div>
          </div>
        );
      })}
    </div>
  );
}
