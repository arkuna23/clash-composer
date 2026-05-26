import { apiRequest } from "./client";
import type { RuleProvider, RuleProviders } from "./types";

const base = (id: string) =>
  `/configs/${encodeURIComponent(id)}/template/rule-providers`;

export function listRuleProviders(
  id: string,
  signal?: AbortSignal,
): Promise<RuleProviders> {
  return apiRequest<RuleProviders>(base(id), { signal });
}

export function getRuleProvider(
  id: string,
  name: string,
  signal?: AbortSignal,
): Promise<RuleProvider> {
  return apiRequest<RuleProvider>(`${base(id)}/${encodeURIComponent(name)}`, {
    signal,
  });
}

export function createRuleProvider(
  id: string,
  name: string,
  body: RuleProvider,
): Promise<RuleProvider> {
  return apiRequest<RuleProvider>(`${base(id)}/${encodeURIComponent(name)}`, {
    method: "POST",
    body,
  });
}

export function updateRuleProvider(
  id: string,
  name: string,
  body: RuleProvider,
): Promise<RuleProvider> {
  return apiRequest<RuleProvider>(`${base(id)}/${encodeURIComponent(name)}`, {
    method: "PUT",
    body,
  });
}

export function deleteRuleProvider(id: string, name: string): Promise<void> {
  return apiRequest<void>(`${base(id)}/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
}
