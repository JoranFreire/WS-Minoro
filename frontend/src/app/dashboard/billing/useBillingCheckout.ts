"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useRouter } from "next/navigation";
import { subscribeToPlan, type Plan } from "@/lib/api";

// The exact shape Pagar.me's tokenizecard.js passes to success() isn't
// fully documented — this defensively checks the couple of shapes their
// docs and examples suggest (a bare token id, or a nested token object).
interface PagarmeTokenResult {
  id?: string;
  token?: { id?: string } | string;
}

declare global {
  interface Window {
    PagarmeCheckout?: {
      init: (
        success: (data: PagarmeTokenResult) => boolean | void,
        fail: (error: unknown) => void
      ) => void;
    };
  }
}

function extractToken(data: PagarmeTokenResult): string | null {
  if (data.id) return data.id;
  if (typeof data.token === "string") return data.token;
  if (data.token?.id) return data.token.id;
  return null;
}

export function useBillingCheckout() {
  const router = useRouter();
  const [plan, setPlan] = useState<Plan>("starter");
  const [error, setError] = useState("");
  const [submitting, setSubmitting] = useState(false);

  // The tokenizecard.js success callback is registered once, when the
  // script loads — it would otherwise close over the plan selected at that
  // moment instead of whatever the user picks before submitting.
  const planRef = useRef(plan);
  useEffect(() => {
    planRef.current = plan;
  }, [plan]);

  const initialized = useRef(false);

  const submitToken = useCallback(
    (cardToken: string) => {
      setSubmitting(true);
      setError("");
      subscribeToPlan(planRef.current, cardToken)
        .then(() => router.push("/dashboard/settings"))
        .catch((err) => {
          const response = (err as { response?: { data?: { error?: string } } }).response;
          setError(response?.data?.error ?? "Could not process the subscription.");
        })
        .finally(() => setSubmitting(false));
    },
    [router]
  );

  const onScriptLoad = useCallback(() => {
    if (initialized.current) return;
    initialized.current = true;

    window.PagarmeCheckout?.init(
      (data) => {
        const token = extractToken(data);
        if (!token) {
          setError("Could not read the card token from Pagar.me's response.");
          return false;
        }
        submitToken(token);
        // Always false: we submit via our own API call (so we control
        // loading/error state and can redirect with next/navigation),
        // never the widget's own raw form POST.
        return false;
      },
      () => setError("Card validation failed. Check the card details and try again.")
    );
  }, [submitToken]);

  return { plan, setPlan, error, submitting, onScriptLoad };
}
