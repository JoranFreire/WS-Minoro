import { useState } from "react";
import { Link } from "@/lib/api";
import { useCreateLink, useUpdateLink, useAddDestination } from "@/hooks/useLinks";

export interface DestinationDraft {
  url: string;
  weight: number;
  max_clicks: string;
}

interface UseLinkFormArgs {
  link?: Link;
  onClose: () => void;
}

export function useLinkForm({ link, onClose }: UseLinkFormArgs) {
  const [title, setTitle] = useState(link?.title || "");
  const [fallbackUrl, setFallbackUrl] = useState(link?.fallback_url || "");
  const [strategy, setStrategy] = useState<Link["routing_strategy"]>(
    link?.routing_strategy || "round_robin"
  );
  const [isActive, setIsActive] = useState(link?.is_active ?? true);
  const [destinations, setDestinations] = useState<DestinationDraft[]>([
    { url: "", weight: 1, max_clicks: "" },
  ]);

  const createLink = useCreateLink();
  const updateLink = useUpdateLink();
  const addDestination = useAddDestination();
  const isEditing = !!link;

  const addRow = () =>
    setDestinations((prev) => [...prev, { url: "", weight: 1, max_clicks: "" }]);

  const removeRow = (i: number) =>
    setDestinations((prev) => prev.filter((_, idx) => idx !== i));

  const updateRow = (i: number, field: keyof DestinationDraft, value: string | number) =>
    setDestinations((prev) =>
      prev.map((d, idx) => (idx === i ? { ...d, [field]: value } : d))
    );

  const handleSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    const data = {
      title,
      fallback_url: fallbackUrl,
      routing_strategy: strategy,
      is_active: isActive,
    };

    if (isEditing && link) {
      await updateLink.mutateAsync({ id: link.id, data });
    } else {
      const created = await createLink.mutateAsync(data);
      const validDestinations = destinations.filter((d) => d.url.trim());
      for (const dest of validDestinations) {
        await addDestination.mutateAsync({
          linkId: created.id,
          data: {
            url: dest.url,
            weight: dest.weight,
            max_clicks: dest.max_clicks ? parseInt(dest.max_clicks) : undefined,
          },
        });
      }
    }
    onClose();
  };

  const isPending =
    createLink.isPending || updateLink.isPending || addDestination.isPending;

  const error = (createLink.error || updateLink.error || addDestination.error) as {
    response?: { data?: { message?: string; error?: string } };
  } | null;

  return {
    isEditing,
    title,
    setTitle,
    fallbackUrl,
    setFallbackUrl,
    strategy,
    setStrategy,
    isActive,
    setIsActive,
    destinations,
    addRow,
    removeRow,
    updateRow,
    handleSubmit,
    isPending,
    error,
  };
}
