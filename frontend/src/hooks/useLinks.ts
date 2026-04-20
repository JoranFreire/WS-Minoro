import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import * as api from "@/lib/api";

export function useLinks() {
  return useQuery({
    queryKey: ["links"],
    queryFn: () => api.listLinks().then((r) => r.data),
  });
}

export function useLink(id: string) {
  return useQuery({
    queryKey: ["links", id],
    queryFn: () => api.getLink(id).then((r) => r.data),
    enabled: !!id,
  });
}

export function useCreateLink() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (data: Partial<api.Link>) => api.createLink(data).then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["links"] }),
  });
}

export function useUpdateLink() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: Partial<api.Link> }) =>
      api.updateLink(id, data).then((r) => r.data),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["links"] }),
  });
}

export function useDeleteLink() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: (id: string) => api.deleteLink(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ["links"] }),
  });
}

export function useAddDestination() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ linkId, data }: { linkId: string; data: Partial<api.Destination> }) =>
      api.addDestination(linkId, data).then((r) => r.data),
    onSuccess: (_d, { linkId }) => qc.invalidateQueries({ queryKey: ["links", linkId] }),
  });
}

export function useDeleteDestination() {
  const qc = useQueryClient();
  return useMutation({
    mutationFn: ({ linkId, destId }: { linkId: string; destId: string }) =>
      api.deleteDestination(linkId, destId),
    onSuccess: (_d, { linkId }) => qc.invalidateQueries({ queryKey: ["links", linkId] }),
  });
}
