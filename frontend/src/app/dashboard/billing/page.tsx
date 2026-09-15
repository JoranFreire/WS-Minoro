"use client";

import Script from "next/script";
import { useBillingCheckout } from "./useBillingCheckout";
import type { Plan } from "@/lib/api";

const PLANS: { id: Plan; label: string; quota: string }[] = [
  { id: "starter", label: "Starter", quota: "50,000 clicks/month" },
  { id: "pro", label: "Pro", quota: "500,000 clicks/month" },
  { id: "business", label: "Business", quota: "5,000,000 clicks/month" },
];

const INPUT =
  "w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500 text-sm text-gray-900 bg-white";

export default function BillingPage() {
  const { plan, setPlan, error, submitting, onScriptLoad } = useBillingCheckout();
  const publicKey = process.env.NEXT_PUBLIC_PAGARME_PUBLIC_KEY;

  if (!publicKey) {
    return (
      <div>
        <h1 className="text-2xl font-bold text-gray-900">Billing</h1>
        <p className="text-gray-500 text-sm mt-2">
          Billing isn&apos;t configured for this deployment yet.
        </p>
      </div>
    );
  }

  const selectedPlan = PLANS.find((p) => p.id === plan);

  return (
    <div className="max-w-lg">
      <div className="mb-6">
        <h1 className="text-2xl font-bold text-gray-900">Upgrade plan</h1>
        <p className="text-gray-500 text-sm mt-0.5">
          Payments processed by Pagar.me — your card details never touch our servers.
        </p>
      </div>

      <Script
        src="https://checkout.pagar.me/v1/tokenizecard.js"
        data-pagarmecheckout-app-id={publicKey}
        onLoad={onScriptLoad}
      />

      <div className="bg-white rounded-xl border border-gray-200 p-5 space-y-4">
        {error && (
          <div className="bg-red-50 border border-red-200 text-red-700 px-3 py-2 rounded-lg text-sm">
            {error}
          </div>
        )}

        <div>
          <label className="block text-sm font-medium text-gray-700 mb-2">Plan</label>
          <div className="grid grid-cols-3 gap-2">
            {PLANS.map((p) => (
              <button
                key={p.id}
                type="button"
                onClick={() => setPlan(p.id)}
                className={`px-3 py-2 rounded-lg text-xs font-medium border transition-colors text-left ${
                  plan === p.id
                    ? "border-green-600 bg-green-50 text-green-700"
                    : "border-gray-300 text-gray-600 hover:bg-gray-50"
                }`}
              >
                <div className="font-semibold">{p.label}</div>
                <div className="text-gray-400 mt-0.5">{p.quota}</div>
              </button>
            ))}
          </div>
        </div>

        <form data-pagarmecheckout-form className="space-y-3">
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Cardholder name</label>
            <input
              name="holder-name"
              data-pagarmecheckout-element="holder_name"
              className={INPUT}
              required
            />
          </div>
          <div>
            <label className="block text-sm font-medium text-gray-700 mb-1">Card number</label>
            <input
              name="card-number"
              data-pagarmecheckout-element="number"
              className={INPUT}
              required
            />
          </div>
          <div className="flex gap-3">
            <div className="flex-1">
              <label className="block text-sm font-medium text-gray-700 mb-1">Month</label>
              <input
                name="card-exp-month"
                data-pagarmecheckout-element="exp_month"
                placeholder="MM"
                className={INPUT}
                required
              />
            </div>
            <div className="flex-1">
              <label className="block text-sm font-medium text-gray-700 mb-1">Year</label>
              <input
                name="card-exp-year"
                data-pagarmecheckout-element="exp_year"
                placeholder="YY"
                className={INPUT}
                required
              />
            </div>
            <div className="flex-1">
              <label className="block text-sm font-medium text-gray-700 mb-1">CVV</label>
              <input name="cvv" data-pagarmecheckout-element="cvv" className={INPUT} required />
            </div>
          </div>
          <button
            type="submit"
            disabled={submitting}
            className="w-full bg-green-600 text-white py-2 px-4 rounded-lg font-medium hover:bg-green-700 disabled:opacity-50 transition-colors"
          >
            {submitting ? "Processing..." : `Subscribe to ${selectedPlan?.label}`}
          </button>
        </form>
      </div>
    </div>
  );
}
