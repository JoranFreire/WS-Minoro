"use client";

import { Link } from "@/lib/api";
import { INPUT_SM } from "./formStyles";

interface AddDestinationFormProps {
  strategy: Link["routing_strategy"];
  url: string;
  weight: number;
  maxClicks: string;
  isPending: boolean;
  onUrlChange: (url: string) => void;
  onWeightChange: (weight: number) => void;
  onMaxClicksChange: (maxClicks: string) => void;
  onCancel: () => void;
  onSubmit: (e: React.FormEvent) => void;
}

export function AddDestinationForm({
  strategy,
  url,
  weight,
  maxClicks,
  isPending,
  onUrlChange,
  onWeightChange,
  onMaxClicksChange,
  onCancel,
  onSubmit,
}: AddDestinationFormProps) {
  return (
    <form onSubmit={onSubmit} className="bg-gray-50 rounded-lg p-3 space-y-2 border border-gray-200">
      <input
        type="url"
        value={url}
        onChange={(e) => onUrlChange(e.target.value)}
        placeholder="https://chat.whatsapp.com/..."
        className={INPUT_SM}
        required
      />
      <div className="flex gap-2">
        {strategy === "weighted" && (
          <div className="flex-1">
            <label className="text-xs text-gray-500 mb-0.5 block">Peso</label>
            <input
              type="number"
              value={weight}
              onChange={(e) => onWeightChange(parseInt(e.target.value))}
              min={1}
              className={INPUT_SM}
            />
          </div>
        )}
        <div className="flex-1">
          <label className="text-xs text-gray-500 mb-0.5 block">Máx. cliques</label>
          <input
            type="number"
            value={maxClicks}
            onChange={(e) => onMaxClicksChange(e.target.value)}
            placeholder="∞"
            className={INPUT_SM}
          />
        </div>
      </div>
      <div className="flex gap-2">
        <button
          type="button"
          onClick={onCancel}
          className="flex-1 px-3 py-1.5 border border-gray-300 text-gray-600 rounded text-xs hover:bg-gray-100"
        >
          Cancelar
        </button>
        <button
          type="submit"
          disabled={isPending}
          className="flex-1 px-3 py-1.5 bg-green-600 text-white rounded text-xs hover:bg-green-700 disabled:opacity-50"
        >
          {isPending ? "Adicionando..." : "Adicionar"}
        </button>
      </div>
    </form>
  );
}
