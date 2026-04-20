import { useQuery } from "@tanstack/react-query";
import api from "@/lib/api";

export interface ClickAggregate {
  period_start: string;
  clicks: number;
}

export interface ClickByCountry {
  country_code: string;
  clicks: number;
}

export interface ClickByDevice {
  device_type: string;
  clicks: number;
}

interface TimeSeriesResponse {
  data: ClickAggregate[];
  from: string;
  to: string;
  granularity: string;
}

interface CountryResponse {
  data: ClickByCountry[];
}

interface DeviceResponse {
  data: ClickByDevice[];
}

export function useClickTimeSeries(
  linkId: string | null,
  from: string,
  to: string,
  granularity: "day" | "hour" = "day"
) {
  return useQuery<TimeSeriesResponse>({
    queryKey: ["analytics", "timeseries", linkId, from, to, granularity],
    queryFn: () =>
      api
        .get(`/api/v1/analytics/links/${linkId}`, {
          params: { from, to, granularity },
        })
        .then((r) => r.data),
    enabled: !!linkId,
    staleTime: 60_000,
  });
}

export function useClicksByCountry(
  linkId: string | null,
  from: string,
  to: string
) {
  return useQuery<CountryResponse>({
    queryKey: ["analytics", "countries", linkId, from, to],
    queryFn: () =>
      api
        .get(`/api/v1/analytics/links/${linkId}/countries`, {
          params: { from, to },
        })
        .then((r) => r.data),
    enabled: !!linkId,
    staleTime: 60_000,
  });
}

export function useClicksByDevice(
  linkId: string | null,
  from: string,
  to: string
) {
  return useQuery<DeviceResponse>({
    queryKey: ["analytics", "devices", linkId, from, to],
    queryFn: () =>
      api
        .get(`/api/v1/analytics/links/${linkId}/devices`, {
          params: { from, to },
        })
        .then((r) => r.data),
    enabled: !!linkId,
    staleTime: 60_000,
  });
}
