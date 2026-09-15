import axios from "axios";
import { clearAuthState } from "./auth";

// Session tokens live in httpOnly cookies set by link-admin, so the browser
// attaches them automatically — the client never sees or handles the token.
const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081",
  headers: { "Content-Type": "application/json" },
  withCredentials: true,
});

// Coalesces concurrent 401s into a single /auth/refresh call instead of a
// stampede — several requests can fail at once right as the access token
// expires.
let refreshPromise: Promise<boolean> | null = null;

function refreshSession(): Promise<boolean> {
  if (!refreshPromise) {
    refreshPromise = api
      .post("/auth/refresh")
      .then(() => true)
      .catch(() => false)
      .finally(() => {
        refreshPromise = null;
      });
  }
  return refreshPromise;
}

interface RetriableConfig {
  url?: string;
  _retry?: boolean;
}

api.interceptors.response.use(
  (res) => res,
  async (err) => {
    const status = err.response?.status;
    const originalRequest = err.config as RetriableConfig | undefined;
    // /auth/* 401s (e.g. wrong password on login) are handled by whoever
    // called them, not globally — otherwise a failed login attempt reloads
    // the page before the caller's own error message is ever shown.
    const isAuthEndpoint = originalRequest?.url?.startsWith("/auth/");

    if (status === 401 && !isAuthEndpoint && originalRequest && !originalRequest._retry) {
      originalRequest._retry = true;
      if (await refreshSession()) {
        return api(originalRequest);
      }
    }

    if (status === 401 && !isAuthEndpoint && typeof window !== "undefined") {
      clearAuthState();
      // A full reload (not useRouter/redirect) is intentional here: this
      // interceptor runs outside React's render tree, and a hard navigation
      // also clears the react-query cache and any other in-memory state
      // left over from the expired session.
      // eslint-disable-next-line @next/next/no-location-assign-relative-destination
      window.location.href = "/login";
    }
    return Promise.reject(err);
  }
);

// --- Auth ---
export const register = (tenantName: string, email: string, password: string) =>
  api.post<{ message: string }>("/auth/register", { tenant_name: tenantName, email, password });
export const login = (email: string, password: string) =>
  api.post<{ message: string }>("/auth/login", { email, password });
export const logout = () => api.post<{ message: string }>("/auth/logout");

// --- Links ---
export interface Link {
  id: string;
  tenant_id: string;
  short_code: string;
  title: string;
  fallback_url: string;
  routing_strategy: "single" | "round_robin" | "weighted";
  is_active: boolean;
  created_at: string;
  updated_at: string;
}

export interface Destination {
  id: string;
  link_id: string;
  url: string;
  weight: number;
  max_clicks: number | null;
  current_clicks: number;
  is_active: boolean;
}

export const listLinks = () => api.get<Link[]>("/api/v1/links");
export const getLink = (id: string) =>
  api.get<{ link: Link; destinations: Destination[] }>(`/api/v1/links/${id}`);
export const createLink = (data: Partial<Link>) => api.post<Link>("/api/v1/links", data);
export const updateLink = (id: string, data: Partial<Link>) =>
  api.put<Link>(`/api/v1/links/${id}`, data);
export const deleteLink = (id: string) => api.delete(`/api/v1/links/${id}`);

export const addDestination = (linkId: string, data: Partial<Destination>) =>
  api.post<Destination>(`/api/v1/links/${linkId}/destinations`, data);
export const updateDestination = (linkId: string, destId: string, data: Partial<Destination>) =>
  api.put(`/api/v1/links/${linkId}/destinations/${destId}`, data);
export const deleteDestination = (linkId: string, destId: string) =>
  api.delete(`/api/v1/links/${linkId}/destinations/${destId}`);

// --- Tenant ---
export interface Tenant {
  id: string;
  name: string;
  plan: string;
  quota_clicks_month: number;
}

export interface QuotaInfo {
  clicks_used: number;
  clicks_limit: number;
  remaining: number;
}

export const getTenant = () => api.get<Tenant>("/api/v1/tenants/me");
export const getQuota = () => api.get<QuotaInfo>("/api/v1/tenants/me/quota");

// --- Billing ---
export type Plan = "starter" | "pro" | "business";

export const subscribeToPlan = (plan: Plan, cardToken: string) =>
  api.post<{ plan: string; quota_clicks_month: number }>("/api/v1/billing/subscribe", {
    plan,
    card_token: cardToken,
  });

export const cancelSubscription = () =>
  api.delete<{ plan: string; quota_clicks_month: number }>("/api/v1/billing/subscription");

export default api;
