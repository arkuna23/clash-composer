// Mirrors composer/merge.go MergeRule and ConfigSource.
export type RulesetStrategy = "url-ruleset" | "replace-ruleset" | "";

export interface ConfigSource {
  path?: string;
  url?: string;
  cmd?: string;
}

export interface ConfigGroup {
  sources?: ConfigSource[];
  includeDirect?: boolean;
  includeGroups?: string[];
}

export interface MergeRule {
  template: string;
  configurations: Record<string, ConfigGroup | ConfigSource[]>;
  rulesetStrategy?: RulesetStrategy;
}

export interface ConfigListResponse {
  configs: string[];
}

// Generic provider object: rule-providers entries are arbitrary YAML maps.
export type RuleProvider = Record<string, unknown>;

export type RuleProviders = Record<string, RuleProvider>;

export interface RuleGroup {
  name: string;
  rules: string[];
}

export interface CreateRuleGroupPayload {
  name: string;
  index: number;
  rules: string[];
}

export interface UpdateRuleGroupPayload {
  name?: string;
  rules?: string[];
}

export interface AddRulePayload {
  rule: string;
  index?: number;
}

export interface UpdateRulePayload {
  rule: string;
}

export interface ApiErrorBody {
  error: string;
}
