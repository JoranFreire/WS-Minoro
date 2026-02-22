"use client";

import { useState } from "react";
import { useQuota } from "@/hooks/useQuota";
import { useLinks } from "@/hooks/useLinks";
import {
  useClickTimeSeries,
  useClicksByCountry,
  useClicksByDevice,
} from "@/hooks/useAnalytics";
import { ClickChart } from "@/components/analytics/ClickChart";
import { GeoBreakdown } from "@/components/analytics/GeoBreakdown";

function toDateStr(d: Date) {
  return d.toISOString().split("T")[0];
}

const DEFAULT_TO = toDateStr(new Date());
const DEFAULT_FROM = toDateStr(new Date(Date.now() - 30 * 24 * 60 * 60 * 1000));

export default function AnalyticsPage() {
  const { data: quota } = useQuota();
  const { data: links } = useLinks();

  const [selectedLinkId, setSelectedLinkId] = useState<string>("");
  const [granularity, setGranularity] = useState<"day" | "hour">("day");

  const linkId = selectedLinkId || null;

  const { data: timeSeries, isLoading: tsLoading } = useClickTimeSeries(
    linkId,
    DEFAULT_FROM,
    DEFAULT_TO,
    granularity
  );
  const { data: countries } = useClicksByCountry(linkId, DEFAULT_FROM, DEFAULT_TO);
  const { data: devices } = useClicksByDevice(linkId, DEFAULT_FROM, DEFAULT_TO);

  const usedPercent = quota
    ? Math.round((quota.clicks_used / quota.clicks_limit) * 100)
    : 0;

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Analytics</h1>
        <p className="text-gray-500 text-sm mt-0.5">Last 30 days</p>
      </div>

      {/* ── KPI cards ───────────────────────────────────────── */}
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

      {/* ── Quota bar ───────────────────────────────────────── */}
      {quota && (
        <div className="bg-white rounded-xl border border-gray-200 p-5 mb-6">
          <div className="flex justify-between text-sm mb-2">
            <span className="font-medium text-gray-700">Monthly quota</span>
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

      {/* ── Link selector + granularity ─────────────────────── */}
      <div className="flex flex-wrap gap-3 mb-4">
        <select
          value={selectedLinkId}
          onChange={(e) => setSelectedLinkId(e.target.value)}
          className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-green-500"
        >
          <option value="">— Select a link —</option>
          {links?.map((l) => (
            <option key={l.id} value={l.id}>
              {l.title || l.short_code}
            </option>
          ))}
        </select>

        <select
          value={granularity}
          onChange={(e) => setGranularity(e.target.value as "day" | "hour")}
          className="px-3 py-2 border border-gray-300 rounded-lg text-sm focus:outline-none focus:ring-2 focus:ring-green-500"
        >
          <option value="day">Daily</option>
          <option value="hour">Hourly</option>
        </select>
      </div>

      {/* ── Click time-series chart ─────────────────────────── */}
      <div className="bg-white rounded-xl border border-gray-200 p-5 mb-6">
        <p className="text-sm font-medium text-gray-700 mb-4">Clicks over time</p>
        {!linkId ? (
          <p className="text-sm text-gray-400 py-6 text-center">Select a link to view chart</p>
        ) : tsLoading ? (
          <div className="h-[220px] flex items-center justify-center">
            <span className="text-sm text-gray-400">Loading…</span>
          </div>
        ) : (
          <ClickChart
            data={timeSeries?.data ?? []}
            granularity={granularity}
          />
        )}
      </div>

      {/* ── Country + Device breakdown ──────────────────────── */}
      {linkId && (
        <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
          <div className="bg-white rounded-xl border border-gray-200 p-5">
            <p className="text-sm font-medium text-gray-700 mb-4">By country</p>
            <GeoBreakdown data={countries?.data ?? []} />
          </div>

          <div className="bg-white rounded-xl border border-gray-200 p-5">
            <p className="text-sm font-medium text-gray-700 mb-4">By device</p>
            {(devices?.data ?? []).length === 0 ? (
              <p className="text-sm text-gray-400 py-4 text-center">No device data yet</p>
            ) : (
              <div className="space-y-2">
                {devices?.data.map((row) => (
                  <div key={row.device_type} className="flex justify-between text-sm py-1 border-b border-gray-50 last:border-0">
                    <span className="text-gray-700 capitalize">{row.device_type}</span>
                    <span className="font-medium text-gray-900">{row.clicks.toLocaleString()}</span>
                  </div>
                ))}
              </div>
            )}
          </div>
        </div>
      )}
    </div>
  );
}
