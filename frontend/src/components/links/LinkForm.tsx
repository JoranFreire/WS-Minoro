"use client";

import { useState } from "react";
import { Link } from "@/lib/api";
import { useCreateLink, useUpdateLink, useAddDestination } from "@/hooks/useLinks";
import { Plus, Trash2 } from "lucide-react";

interface DestinationDraft {
  url: string;
  weight: number;
  max_clicks: string;
}

interface LinkFormProps {
  link?: Link;
  onClose: () => void;
}

export function LinkForm({ link, onClose }: LinkFormProps) {
  const [title, setTitle] = useState(link?.title || "");
  const [fallbackUrl, setFallbackUrl] = useState(link?.fallback_url || "");
  const [strategy, setStrategy] = useState<"single" | "round_robin" | "weighted">(
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
      routing_strategy: strategy as Link["routing_strategy"],
      is_active: isActive,
    };

    if (isEditing) {
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

  return (
    <form onSubmit={handleSubmit} className="space-y-4">
      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Título</label>
        <input
          type="text"
          value={title}
          onChange={(e) => setTitle(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
          placeholder="Grupo WhatsApp Vendas"
          required
        />
      </div>

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          Estratégia de roteamento
        </label>
        <select
          value={strategy}
          onChange={(e) => setStrategy(e.target.value as "single" | "round_robin" | "weighted")}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
        >
          <option value="round_robin">Round Robin — distribuir igualmente</option>
          <option value="weighted">Weighted — distribuir por peso</option>
          <option value="single">Single — sempre o primeiro ativo</option>
        </select>
      </div>

      {!isEditing && (
        <div>
          <div className="flex items-center justify-between mb-2">
            <label className="block text-sm font-medium text-gray-700">
              Links de destino
            </label>
            <button
              type="button"
              onClick={addRow}
              className="flex items-center gap-1 text-xs text-green-700 hover:text-green-800 font-medium"
            >
              <Plus className="w-3.5 h-3.5" />
              Adicionar
            </button>
          </div>
          <div className="space-y-2">
            {destinations.map((dest, i) => (
              <div key={i} className="flex gap-2 items-start">
                <div className="flex-1 space-y-1">
                  <input
                    type="url"
                    value={dest.url}
                    onChange={(e) => updateRow(i, "url", e.target.value)}
                    placeholder="https://chat.whatsapp.com/..."
                    className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
                  />
                  {strategy !== "single" && (
                    <div className="flex gap-2">
                      {strategy === "weighted" && (
                        <div className="flex-1">
                          <input
                            type="number"
                            value={dest.weight}
                            onChange={(e) => updateRow(i, "weight", parseInt(e.target.value))}
                            min={1}
                            placeholder="Peso"
                            className="w-full px-2 py-1.5 border border-gray-300 rounded text-xs focus:outline-none focus:ring-2 focus:ring-green-500"
                          />
                        </div>
                      )}
                      <div className="flex-1">
                        <input
                          type="number"
                          value={dest.max_clicks}
                          onChange={(e) => updateRow(i, "max_clicks", e.target.value)}
                          placeholder="Máx. cliques (opcional)"
                          className="w-full px-2 py-1.5 border border-gray-300 rounded text-xs focus:outline-none focus:ring-2 focus:ring-green-500"
                        />
                      </div>
                    </div>
                  )}
                </div>
                {destinations.length > 1 && (
                  <button
                    type="button"
                    onClick={() => removeRow(i)}
                    className="mt-2 p-1 text-gray-400 hover:text-red-500"
                  >
                    <Trash2 className="w-4 h-4" />
                  </button>
                )}
              </div>
            ))}
          </div>
        </div>
      )}

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          URL de fallback <span className="text-gray-400 font-normal text-xs">(quando todos os destinos estiverem cheios)</span>
        </label>
        <input
          type="url"
          value={fallbackUrl}
          onChange={(e) => setFallbackUrl(e.target.value)}
          className="w-full px-3 py-2 border border-gray-300 rounded-lg focus:outline-none focus:ring-2 focus:ring-green-500 text-sm"
          placeholder="https://..."
        />
      </div>

      {isEditing && (
        <div className="flex items-center gap-3">
          <span className="text-sm font-medium text-gray-700">Ativo</span>
          <button
            type="button"
            onClick={() => setIsActive(!isActive)}
            className={`relative inline-flex h-6 w-11 items-center rounded-full transition-colors ${
              isActive ? "bg-green-600" : "bg-gray-200"
            }`}
          >
            <span
              className={`inline-block h-4 w-4 transform rounded-full bg-white transition-transform ${
                isActive ? "translate-x-6" : "translate-x-1"
              }`}
            />
          </button>
        </div>
      )}

      <div className="flex gap-3 pt-2">
        <button
          type="button"
          onClick={onClose}
          className="flex-1 px-4 py-2 border border-gray-300 text-gray-700 rounded-lg hover:bg-gray-50 text-sm font-medium transition-colors"
        >
          Cancelar
        </button>
        <button
          type="submit"
          disabled={isPending}
          className="flex-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 text-sm font-medium transition-colors"
        >
          {isPending ? "Salvando..." : isEditing ? "Atualizar" : "Criar"}
        </button>
      </div>
    </form>
  );
}
