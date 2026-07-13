// Mirrors composer/merge.go MergeRule and ConfigSource.
export type RulesetStrategy = "url-ruleset" | "replace-ruleset" | "";

export interface ConfigSource {
  path?: string;
  url?: string;
  cmd?: string;
}

export interface ConfigGroup {
  name: string;
  sources?: ConfigSource[];
  includeDirect?: boolean;
  includeGroups?: string[];
  enableUrlTest?: boolean;
}

export interface MergeRule {
  template: string;
  configurations: ConfigGroup[];
  rulesetStrategy?: RulesetStrategy;
  cacheDurationSeconds?: number;
}

export interface ConfigListResponse {
  configs: string[];
}

export interface UploadFileResponse {
  path: string;
  size: number;
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
  rules?: string[];
  rulesYaml?: string;
}

export interface UpdateRuleGroupPayload {
  name?: string;
  rules?: string[];
}

export interface AddRulePayload {
  rule?: string;
  rules?: string[];
  rulesYaml?: string;
  index?: number;
}

export interface UpdateRulePayload {
  rule: string;
}

export interface ApiErrorBody {
  error: string;
}
