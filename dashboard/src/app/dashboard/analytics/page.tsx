"use client";

import { useQuota } from "@/hooks/useQuota";
import { useLinks } from "@/hooks/useLinks";

export default function AnalyticsPage() {
  const { data: quota } = useQuota();
  const { data: links } = useLinks();

  const usedPercent = quota ? Math.round((quota.clicks_used / quota.clicks_limit) * 100) : 0;

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Analytics</h1>
        <p className="text-gray-500 text-sm mt-0.5">This month&apos;s performance</p>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-3 gap-4 mb-8">
        <div className="bg-white rounded-xl border border-gray-200 p-5">
          <p className="text-sm text-gray-500">Total links</p>
          <p className="text-3xl font-bold text-gray-900 mt-1">{links?.length ?? 0}</p>
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-5">
          <p className="text-sm text-gray-500">Clicks this month</p>
          <p className="text-3xl font-bold text-gray-900 mt-1">
            {quota?.clicks_used?.toLocaleString() ?? "–"}
          </p>
        </div>
        <div className="bg-white rounded-xl border border-gray-200 p-5">
          <p className="text-sm text-gray-500">Quota remaining</p>
          <p className="text-3xl font-bold text-gray-900 mt-1">
            {quota?.remaining?.toLocaleString() ?? "–"}
          </p>
        </div>
      </div>

      {quota && (
        <div className="bg-white rounded-xl border border-gray-200 p-5 mb-6">
          <div className="flex justify-between text-sm mb-2">
            <span className="font-medium text-gray-700">Monthly quota usage</span>
            <span className="text-gray-500">{usedPercent}%</span>
          </div>
          <div className="h-3 bg-gray-100 rounded-full overflow-hidden">
            <div
              className={`h-full rounded-full transition-all ${
                usedPercent > 90
                  ? "bg-red-500"
                  : usedPercent > 70
                  ? "bg-yellow-500"
                  : "bg-green-500"
              }`}
              style={{ width: `${Math.min(usedPercent, 100)}%` }}
            />
          </div>
          <p className="text-xs text-gray-400 mt-2">
            {quota.clicks_used.toLocaleString()} / {quota.clicks_limit.toLocaleString()} clicks
          </p>
        </div>
      )}

      <div className="bg-white rounded-xl border border-gray-200 p-5">
        <p className="text-sm font-medium text-gray-700 mb-4">Active links</p>
        <div className="divide-y divide-gray-100">
          {links?.filter((l) => l.is_active).map((link) => (
            <div key={link.id} className="py-3 flex items-center justify-between">
              <div>
                <p className="text-sm font-medium text-gray-800">{link.title || link.short_code}</p>
                <p className="text-xs text-gray-400 font-mono">{link.short_code}</p>
              </div>
              <span className="text-xs bg-blue-50 text-blue-700 px-2 py-0.5 rounded font-medium">
                {link.routing_strategy.replace("_", " ")}
              </span>
            </div>
          ))}
          {(!links || links.filter((l) => l.is_active).length === 0) && (
            <p className="text-sm text-gray-400 py-4 text-center">No active links</p>
          )}
        </div>
      </div>
    </div>
  );
}
