"use client";

import { Link } from "@/lib/api";
import { useLinkForm } from "./useLinkForm";
import { ShortUrlBadge } from "./ShortUrlBadge";
import { RoutingStrategySelect } from "./RoutingStrategySelect";
import { DestinationDraftEditor } from "./DestinationDraftEditor";
import { ActiveToggle } from "./ActiveToggle";
import { INPUT } from "./formStyles";

interface LinkFormProps {
  link?: Link;
  onClose: () => void;
}

export function LinkForm({ link, onClose }: LinkFormProps) {
  const form = useLinkForm({ link, onClose });

  const routerBase =
    process.env.NEXT_PUBLIC_ROUTER_URL?.replace(/\/$/, "") || "http://localhost:8080";
  const shortUrl = form.isEditing && link ? `${routerBase}/${link.short_code}` : null;

  return (
    <form onSubmit={form.handleSubmit} className="space-y-4">
      {shortUrl && <ShortUrlBadge shortUrl={shortUrl} />}

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">Título</label>
        <input
          type="text"
          value={form.title}
          onChange={(e) => form.setTitle(e.target.value)}
          className={INPUT}
          placeholder="Grupo WhatsApp Vendas"
          required
        />
      </div>

      <RoutingStrategySelect value={form.strategy} onChange={form.setStrategy} />

      {!form.isEditing && (
        <DestinationDraftEditor
          strategy={form.strategy}
          destinations={form.destinations}
          onAdd={form.addRow}
          onRemove={form.removeRow}
          onUpdate={form.updateRow}
        />
      )}

      <div>
        <label className="block text-sm font-medium text-gray-700 mb-1">
          URL de fallback{" "}
          <span className="text-gray-400 font-normal text-xs">
            (quando todos os destinos estiverem cheios)
          </span>
        </label>
        <input
          type="url"
          value={form.fallbackUrl}
          onChange={(e) => form.setFallbackUrl(e.target.value)}
          className={INPUT}
          placeholder="https://..."
        />
      </div>

      {form.isEditing && (
        <ActiveToggle active={form.isActive} onToggle={() => form.setIsActive(!form.isActive)} />
      )}

      {form.error && (
        <div className="bg-red-50 border border-red-200 text-red-700 px-3 py-2 rounded-lg text-sm">
          {form.error?.response?.data?.message ??
            form.error?.response?.data?.error ??
            "Erro ao salvar. Verifique os dados e tente novamente."}
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
          disabled={form.isPending}
          className="flex-1 px-4 py-2 bg-green-600 text-white rounded-lg hover:bg-green-700 disabled:opacity-50 text-sm font-medium transition-colors"
        >
          {form.isPending ? "Salvando..." : form.isEditing ? "Atualizar" : "Criar"}
        </button>
      </div>
    </form>
  );
}
