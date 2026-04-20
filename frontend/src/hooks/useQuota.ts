import { useQuery } from "@tanstack/react-query";
import * as api from "@/lib/api";

export function useQuota() {
  return useQuery({
    queryKey: ["quota"],
    queryFn: () => api.getQuota().then((r) => r.data),
  });
}

export function useTenant() {
  return useQuery({
    queryKey: ["tenant"],
    queryFn: () => api.getTenant().then((r) => r.data),
  });
}
