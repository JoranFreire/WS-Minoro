import axios from "axios";

const api = axios.create({
  baseURL: process.env.NEXT_PUBLIC_API_URL || "http://localhost:8081",
  headers: { "Content-Type": "application/json" },
});

api.interceptors.request.use((config) => {
  if (typeof window !== "undefined") {
    const token = localStorage.getItem("access_token");
    if (token) config.headers.Authorization = `Bearer ${token}`;
  }
  return config;
});

api.interceptors.response.use(
  (res) => res,
  (err) => {
    if (err.response?.status === 401 && typeof window !== "undefined") {
      localStorage.removeItem("access_token");
      window.location.href = "/login";
    }
    return Promise.reject(err);
  }
);

// --- Auth ---
export const login = (email: string, password: string) =>
  api.post<{ access_token: string; refresh_token: string }>("/auth/login", { email, password });

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

export default api;
