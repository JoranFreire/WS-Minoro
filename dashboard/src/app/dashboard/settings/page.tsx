"use client";

import { useTenant } from "@/hooks/useQuota";

export default function SettingsPage() {
  const { data: tenant } = useTenant();

  return (
    <div>
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Settings</h1>
        <p className="text-gray-500 text-sm mt-0.5">Manage your account</p>
      </div>

      <div className="bg-white rounded-xl border border-gray-200 p-5 max-w-lg">
        <h2 className="text-sm font-semibold text-gray-700 mb-4">Tenant Info</h2>
        <dl className="space-y-3">
          <div className="flex justify-between text-sm">
            <dt className="text-gray-500">Name</dt>
            <dd className="font-medium text-gray-900">{tenant?.name ?? "–"}</dd>
          </div>
          <div className="flex justify-between text-sm">
            <dt className="text-gray-500">Plan</dt>
            <dd className="font-medium text-gray-900 capitalize">{tenant?.plan ?? "–"}</dd>
          </div>
          <div className="flex justify-between text-sm">
            <dt className="text-gray-500">Monthly click quota</dt>
            <dd className="font-medium text-gray-900">
              {tenant?.quota_clicks_month?.toLocaleString() ?? "–"}
            </dd>
          </div>
          <div className="flex justify-between text-sm">
            <dt className="text-gray-500">Tenant ID</dt>
            <dd className="font-mono text-xs text-gray-500">{tenant?.id ?? "–"}</dd>
          </div>
        </dl>
      </div>
    </div>
  );
}
