import { apiRequest } from "./client";
import type { ConfigListResponse, MergeRule } from "./types";

export function listConfigs(signal?: AbortSignal): Promise<ConfigListResponse> {
  return apiRequest<ConfigListResponse>("/configs", { signal });
}

export function getConfig(id: string, signal?: AbortSignal): Promise<MergeRule> {
  return apiRequest<MergeRule>(`/configs/${encodeURIComponent(id)}`, { signal });
}

export function createConfig(id: string, rule: MergeRule): Promise<MergeRule> {
  return apiRequest<MergeRule>(`/configs/${encodeURIComponent(id)}`, {
    method: "POST",
    body: rule,
  });
}

export function updateConfig(id: string, rule: MergeRule): Promise<MergeRule> {
  return apiRequest<MergeRule>(`/configs/${encodeURIComponent(id)}`, {
    method: "PUT",
    body: rule,
  });
}

export function deleteConfig(id: string): Promise<void> {
  return apiRequest<void>(`/configs/${encodeURIComponent(id)}`, {
    method: "DELETE",
  });
}
