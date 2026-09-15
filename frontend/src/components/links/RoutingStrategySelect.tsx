"use client";

import { ChevronDown } from "lucide-react";
import { Link } from "@/lib/api";
import { INPUT } from "./formStyles";

interface RoutingStrategySelectProps {
  value: Link["routing_strategy"];
  onChange: (strategy: Link["routing_strategy"]) => void;
}

export function RoutingStrategySelect({ value, onChange }: RoutingStrategySelectProps) {
  return (
    <div>
      <label className="block text-sm font-medium text-gray-700 mb-1">
        Estratégia de roteamento
      </label>
      <div className="relative">
        <select
          value={value}
          onChange={(e) => onChange(e.target.value as Link["routing_strategy"])}
          className={`${INPUT} appearance-none pr-10 cursor-pointer`}
        >
          <option value="round_robin">Round Robin — distribuir igualmente</option>
          <option value="weighted">Weighted — distribuir por peso</option>
          <option value="single">Single — sempre o primeiro ativo</option>
        </select>
        <ChevronDown className="absolute right-3 top-1/2 -translate-y-1/2 w-4 h-4 text-gray-400 pointer-events-none" />
      </div>
    </div>
  );
}
