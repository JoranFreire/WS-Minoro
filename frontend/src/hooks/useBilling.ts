import { useMutation, useQueryClient } from "@tanstack/react-query";
import * as api from "@/lib/api";

export function useCancelSubscription() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: () => api.cancelSubscription().then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["tenant"] }),
  });
}
