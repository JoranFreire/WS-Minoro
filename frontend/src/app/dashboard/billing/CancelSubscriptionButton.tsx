"use client";

import { useState } from "react";
import { useCancelSubscription } from "@/hooks/useBilling";

export function CancelSubscriptionButton() {
  const [confirming, setConfirming] = useState(false);
  const cancel = useCancelSubscription();

  if (cancel.isSuccess) {
    return (
      <p className="text-sm text-gray-500">
        Subscription canceled — you&apos;re back on the free plan.
      </p>
    );
  }

  if (!confirming) {
    return (
      <button
        type="button"
        onClick={() => setConfirming(true)}
        className="text-sm text-red-600 hover:text-red-700 font-medium"
      >
        Cancel subscription
      </button>
    );
  }

  return (
    <div className="bg-red-50 border border-red-200 rounded-lg p-3 space-y-2">
      <p className="text-sm text-red-700">
        This cancels your subscription immediately and moves you to the free plan. Are you sure?
      </p>
      {cancel.isError && (
        <p className="text-xs text-red-600">Could not cancel the subscription. Please try again.</p>
      )}
      <div className="flex gap-2">
        <button
          type="button"
          onClick={() => setConfirming(false)}
          disabled={cancel.isPending}
          className="px-3 py-1.5 border border-gray-300 text-gray-600 rounded text-xs hover:bg-gray-100 disabled:opacity-50"
        >
          Keep subscription
        </button>
        <button
          type="button"
          onClick={() => cancel.mutate()}
          disabled={cancel.isPending}
          className="px-3 py-1.5 bg-red-600 text-white rounded text-xs hover:bg-red-700 disabled:opacity-50"
        >
          {cancel.isPending ? "Canceling..." : "Yes, cancel"}
        </button>
      </div>
    </div>
  );
}
