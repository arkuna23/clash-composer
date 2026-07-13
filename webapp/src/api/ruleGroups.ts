import { apiRequest } from "./client";
import type {
  AddRulePayload,
  CreateRuleGroupPayload,
  RuleGroup,
  UpdateRuleGroupPayload,
  UpdateRulePayload,
} from "./types";

const groupBase = (id: string) =>
  `/configs/${encodeURIComponent(id)}/template/rule-groups`;

export function listProxyGroupTargets(
  id: string,
  signal?: AbortSignal,
): Promise<string[]> {
  return apiRequest<string[]>(
    `/configs/${encodeURIComponent(id)}/template/proxy-groups`,
    { signal },
  );
}

export function listRuleGroups(
  id: string,
  signal?: AbortSignal,
): Promise<RuleGroup[]> {
  return apiRequest<RuleGroup[]>(groupBase(id), { signal });
}

export function getRuleGroup(
  id: string,
  name: string,
  signal?: AbortSignal,
): Promise<RuleGroup> {
  return apiRequest<RuleGroup>(`${groupBase(id)}/${encodeURIComponent(name)}`, {
    signal,
  });
}

export function createRuleGroup(
  id: string,
  payload: CreateRuleGroupPayload,
): Promise<RuleGroup> {
  return apiRequest<RuleGroup>(groupBase(id), {
    method: "POST",
    body: payload,
  });
}

export function updateRuleGroup(
  id: string,
  name: string,
  payload: UpdateRuleGroupPayload,
): Promise<RuleGroup> {
  return apiRequest<RuleGroup>(`${groupBase(id)}/${encodeURIComponent(name)}`, {
    method: "PUT",
    body: payload,
  });
}

export function deleteRuleGroup(id: string, name: string): Promise<void> {
  return apiRequest<void>(`${groupBase(id)}/${encodeURIComponent(name)}`, {
    method: "DELETE",
  });
}

const ruleBase = (id: string, groupName: string) =>
  `${groupBase(id)}/${encodeURIComponent(groupName)}/rules`;

export function addRule(
  id: string,
  groupName: string,
  payload: AddRulePayload,
): Promise<RuleGroup> {
  return apiRequest<RuleGroup>(ruleBase(id, groupName), {
    method: "POST",
    body: payload,
  });
}

export function updateRule(
  id: string,
  groupName: string,
  index: number,
  payload: UpdateRulePayload,
): Promise<RuleGroup> {
  return apiRequest<RuleGroup>(`${ruleBase(id, groupName)}/${index}`, {
    method: "PUT",
    body: payload,
  });
}

export function deleteRule(
  id: string,
  groupName: string,
  index: number,
): Promise<RuleGroup> {
  return apiRequest<RuleGroup>(`${ruleBase(id, groupName)}/${index}`, {
    method: "DELETE",
  });
}
