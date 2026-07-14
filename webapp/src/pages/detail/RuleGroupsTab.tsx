import { FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronRight, Pencil, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { cn } from "@/lib/utils";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { CollapsiblePane } from "@/components/ui/collapsible-pane";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { FormError } from "@/components/ui/form-error";
import {
  addRule,
  createRuleGroup,
  deleteRule,
  deleteRuleGroup,
  listRuleGroups,
  listProxyGroupTargets,
  updateRule,
  updateRuleGroup,
} from "@/api/ruleGroups";
import type { RuleGroup } from "@/api/types";
import type { AddRulePayload } from "@/api/types";
import {
  Select,
  SelectContent,
  SelectGroup,
  SelectItem,
  SelectLabel,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Tabs,
  TabsContent,
  TabsList,
  TabsTrigger,
} from "@/components/ui/tabs";

const DEFAULT_GROUP = "default";
const CUSTOM_TARGET = "__custom";
type RuleValueMode = "values" | "conditions" | "none";
type RuleTargetMode = "policy" | "subRule";

function defineRuleType<const Value extends string>(
  value: Value,
  placeholder: string,
  valueMode: RuleValueMode = "values",
  targetMode: RuleTargetMode = "policy",
) {
  return { value, placeholder, valueMode, targetMode };
}

const RULE_TYPE_GROUPS = [
  {
    labelKey: "groups.ruleCategoryDomain",
    types: [
      defineRuleType("DOMAIN", "example.com"),
      defineRuleType("DOMAIN-SUFFIX", "example.com"),
      defineRuleType("DOMAIN-KEYWORD", "google"),
      defineRuleType("DOMAIN-WILDCARD", "*.example.com"),
      defineRuleType("DOMAIN-REGEX", "^api\\..*\\.com$"),
      defineRuleType("GEOSITE", "youtube"),
    ],
  },
  {
    labelKey: "groups.ruleCategoryDestinationIp",
    types: [
      defineRuleType("IP-CIDR", "192.0.2.0/24"),
      defineRuleType("IP-CIDR6", "2001:db8::/32"),
      defineRuleType("IP-SUFFIX", "8.8.8.8/24"),
      defineRuleType("IP-ASN", "13335"),
      defineRuleType("GEOIP", "CN"),
    ],
  },
  {
    labelKey: "groups.ruleCategorySourceIp",
    types: [
      defineRuleType("SRC-GEOIP", "CN"),
      defineRuleType("SRC-IP-ASN", "9808"),
      defineRuleType("SRC-IP-CIDR", "192.168.1.0/24"),
      defineRuleType("SRC-IP-SUFFIX", "192.168.1.201/8"),
    ],
  },
  {
    labelKey: "groups.ruleCategoryPort",
    types: [
      defineRuleType("DST-PORT", "443"),
      defineRuleType("SRC-PORT", "7777"),
    ],
  },
  {
    labelKey: "groups.ruleCategoryInbound",
    types: [
      defineRuleType("IN-PORT", "7890"),
      defineRuleType("IN-TYPE", "SOCKS/HTTP"),
      defineRuleType("IN-USER", "mihomo"),
      defineRuleType("IN-NAME", "mixed-in"),
    ],
  },
  {
    labelKey: "groups.ruleCategoryProcess",
    types: [
      defineRuleType("PROCESS-PATH", "/usr/bin/wget"),
      defineRuleType("PROCESS-PATH-WILDCARD", "/usr/*/wget"),
      defineRuleType("PROCESS-PATH-REGEX", ".*bin/wget"),
      defineRuleType("PROCESS-NAME", "curl"),
      defineRuleType("PROCESS-NAME-WILDCARD", "*telegram*"),
      defineRuleType("PROCESS-NAME-REGEX", "(?i)Telegram"),
      defineRuleType("UID", "1001"),
    ],
  },
  {
    labelKey: "groups.ruleCategoryNetwork",
    types: [
      defineRuleType("NETWORK", "udp"),
      defineRuleType("DSCP", "4"),
    ],
  },
  {
    labelKey: "groups.ruleCategorySpecial",
    types: [
      defineRuleType("RULE-SET", "provider-name"),
      defineRuleType(
        "AND",
        "((DOMAIN,example.com),(NETWORK,UDP))",
        "conditions",
      ),
      defineRuleType(
        "OR",
        "((NETWORK,UDP),(DOMAIN,example.com))",
        "conditions",
      ),
      defineRuleType("NOT", "((DOMAIN,example.com))", "conditions"),
      defineRuleType("SUB-RULE", "(NETWORK,tcp)", "conditions", "subRule"),
      defineRuleType("MATCH", "", "none"),
    ],
  },
] as const;

type RuleTypeGroup = (typeof RULE_TYPE_GROUPS)[number];
type RuleTypeDefinition = RuleTypeGroup["types"][number];
type RuleType = RuleTypeDefinition["value"];

function getRuleTypeDefinition(ruleType: RuleType): RuleTypeDefinition {
  for (const group of RULE_TYPE_GROUPS) {
    const definition = group.types.find((type) => type.value === ruleType);
    if (definition) return definition;
  }
  throw new Error(`unknown rule type: ${ruleType}`);
}

interface RuleBatchDraft {
  mode: "yaml" | "generate";
  yaml: string;
  ruleType: RuleType;
  values: string;
  target: string;
  customTarget: string;
  subRule: string;
}

function emptyRuleBatch(targets: string[]): RuleBatchDraft {
  return {
    mode: "yaml",
    yaml: "",
    ruleType: "DOMAIN-SUFFIX",
    values: "",
    target: targets[0] ?? CUSTOM_TARGET,
    customTarget: "",
    subRule: "",
  };
}

function ruleBatchPayload(draft: RuleBatchDraft): AddRulePayload | null {
  if (draft.mode === "yaml") {
    return draft.yaml.trim() ? { rulesYaml: draft.yaml } : null;
  }
  const definition = getRuleTypeDefinition(draft.ruleType);
  const target =
    definition.targetMode === "subRule"
      ? draft.subRule.trim()
      : draft.target === CUSTOM_TARGET
        ? draft.customTarget.trim()
        : draft.target;
  if (!target) return null;
  if (definition.valueMode === "none") {
    return { rules: [`${draft.ruleType},${target}`] };
  }
  const values = draft.values
    .split("\n")
    .map((value) => value.trim().replace(/^\-\s*/, ""))
    .filter(Boolean);
  if (values.length === 0) return null;
  return {
    rules: values.map((value) => `${draft.ruleType},${value},${target}`),
  };
}

interface RuleGroupsTabProps {
  id: string;
}

export function RuleGroupsTab({ id }: RuleGroupsTabProps) {
  const { t, i18n } = useTranslation();
  const queryClient = useQueryClient();
  const listKey = ["configs", id, "rule-groups"];
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(
    () => new Set(),
  );
  const [createGroupOpen, setCreateGroupOpen] = useState(false);
  const [renameGroup, setRenameGroup] = useState<string | null>(null);
  const [addRuleGroup, setAddRuleGroup] = useState<RuleGroup | null>(null);
  const [editRuleState, setEditRuleState] = useState<{
    group: RuleGroup;
    index: number;
    initial: string;
  } | null>(null);

  const groupsQuery = useQuery({
    queryKey: listKey,
    queryFn: ({ signal }) => listRuleGroups(id, signal),
  });
  const targetsQuery = useQuery({
    queryKey: ["configs", id, "proxy-group-targets"],
    queryFn: ({ signal }) => listProxyGroupTargets(id, signal),
  });

  const refresh = () =>
    queryClient.invalidateQueries({ queryKey: listKey });

  const deleteGroupMutation = useMutation({
    mutationFn: (name: string) => deleteRuleGroup(id, name),
    onSuccess: () => {
      void refresh();
    },
    onError: (error: Error) => {
      toast.error(t("errors.generic", { message: error.message }));
    },
  });

  const deleteRuleMutation = useMutation({
    mutationFn: ({ group, index }: { group: string; index: number }) =>
      deleteRule(id, group, index),
    onSuccess: () => {
      void refresh();
    },
    onError: (error: Error) => {
      toast.error(t("errors.generic", { message: error.message }));
    },
  });

  const onDeleteGroup = (name: string) => {
    if (!window.confirm(t("groups.deleteGroupConfirm", { name }))) return;
    deleteGroupMutation.mutate(name);
  };

  const onDeleteRule = (group: string, index: number) => {
    if (!window.confirm(t("groups.deleteRuleConfirm"))) return;
    deleteRuleMutation.mutate({ group, index });
  };

  const toggle = (name: string) => {
    setExpandedGroups((previous) => {
      const next = new Set(previous);
      if (next.has(name)) {
        next.delete(name);
      } else {
        next.add(name);
      }
      return next;
    });
  };

  useEffect(() => {
    setExpandedGroups(new Set());
  }, [id]);

  const groups = groupsQuery.data ?? [];
  const numberFormatter = new Intl.NumberFormat(i18n.resolvedLanguage);

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button
          onClick={() => setCreateGroupOpen(true)}
          className="gap-1.5"
          disabled={groups.length === 0}
        >
          <Plus className="h-4 w-4" aria-hidden />
          {t("groups.addGroup")}
        </Button>
      </div>
      {groupsQuery.isLoading ? (
        <p
          role="status"
          aria-live="polite"
          className="text-sm text-muted-foreground"
        >
          {t("common.loading")}
        </p>
      ) : groupsQuery.isError ? (
        <p role="alert" className="text-sm text-destructive">
          {(groupsQuery.error as Error).message}
        </p>
      ) : groups.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("groups.empty")}</p>
      ) : (
        <div className="space-y-3">
          {groups.map((group, groupIndex) => {
            const isOpen = expandedGroups.has(group.name);
            const isDefault = group.name === DEFAULT_GROUP;
            const contentID = `rule-group-${groupIndex}-rules`;
            return (
              <div
                key={group.name}
                className="rounded-md border bg-card transition-colors hover:border-foreground/20"
              >
                <div className="flex min-w-0 items-center gap-2 px-3 py-2">
                  <button
                    type="button"
                    onClick={() => toggle(group.name)}
                    className="flex min-w-0 items-center gap-2 rounded-sm text-sm font-medium hover:text-foreground/80 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring cursor-pointer"
                    aria-expanded={isOpen}
                    aria-controls={contentID}
                  >
                    <ChevronRight
                      className={cn(
                        "h-4 w-4 transition-transform duration-200 ease-out motion-reduce:transition-none",
                        isOpen && "rotate-90",
                      )}
                      aria-hidden
                    />
                    <span className="truncate" translate="no">
                      {group.name}
                    </span>
                    {isDefault && (
                      <Badge variant="secondary" className="text-[10px]">
                        default
                      </Badge>
                    )}
                  </button>
                  <span className="shrink-0 text-xs tabular-nums text-muted-foreground">
                    {t("groups.ruleCount", {
                      count: group.rules.length,
                      formattedCount: numberFormatter.format(group.rules.length),
                    })}
                  </span>
                  <div className="ml-auto flex shrink-0 gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setAddRuleGroup(group)}
                      className="gap-1.5 px-2 sm:px-3"
                      title={t("groups.addRule")}
                      aria-label={t("groups.addRule")}
                    >
                      <Plus className="h-4 w-4" aria-hidden />
                      <span className="hidden sm:inline">{t("groups.addRule")}</span>
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={t("groups.renameGroup")}
                      onClick={() => setRenameGroup(group.name)}
                      disabled={isDefault}
                    >
                      <Pencil className="h-4 w-4" aria-hidden />
                    </Button>
                    <Button
                      variant="ghost"
                      size="icon"
                      aria-label={t("groups.deleteGroup")}
                      onClick={() => onDeleteGroup(group.name)}
                      disabled={isDefault}
                    >
                      <Trash2 className="h-4 w-4" aria-hidden />
                    </Button>
                  </div>
                </div>
                <CollapsiblePane id={contentID} open={isOpen}>
                  <div className="border-t px-3 py-2 space-y-1">
                    {group.rules.length === 0 ? (
                      <p className="text-xs text-muted-foreground">
                        {t("groups.empty")}
                      </p>
                    ) : (
                      group.rules.map((rule, index) => (
                        <div
                          key={`${group.name}-${index}`}
                          className="flex items-center gap-2 rounded-md px-2 py-1.5 transition-colors hover:bg-muted/50 [content-visibility:auto] [contain-intrinsic-size:0_40px]"
                        >
                          <span className="text-xs text-muted-foreground w-8 shrink-0 font-mono">
                            {index}
                          </span>
                          <code
                            className="text-xs flex-1 break-all"
                            translate="no"
                          >
                            {rule}
                          </code>
                          <Button
                            variant="ghost"
                            size="icon"
                            aria-label={t("groups.editRule")}
                            onClick={() =>
                              setEditRuleState({
                                group,
                                index,
                                initial: rule,
                              })
                            }
                          >
                            <Pencil className="h-3.5 w-3.5" aria-hidden />
                          </Button>
                          <Button
                            variant="ghost"
                            size="icon"
                            aria-label={t("common.delete")}
                            onClick={() => onDeleteRule(group.name, index)}
                          >
                            <Trash2 className="h-3.5 w-3.5" aria-hidden />
                          </Button>
                        </div>
                      ))
                    )}
                  </div>
                </CollapsiblePane>
              </div>
            );
          })}
        </div>
      )}

      <CreateGroupDialog
        configId={id}
        open={createGroupOpen}
        onOpenChange={setCreateGroupOpen}
        groupCount={groups.length}
        targets={targetsQuery.data ?? []}
        onSuccess={refresh}
      />
      <RenameGroupDialog
        configId={id}
        open={renameGroup !== null}
        onOpenChange={(open) => !open && setRenameGroup(null)}
        currentName={renameGroup ?? ""}
        onSuccess={refresh}
      />
      <AddRuleDialog
        configId={id}
        open={addRuleGroup !== null}
        onOpenChange={(open) => !open && setAddRuleGroup(null)}
        group={addRuleGroup}
        targets={targetsQuery.data ?? []}
        onSuccess={refresh}
      />
      <EditRuleDialog
        configId={id}
        open={editRuleState !== null}
        onOpenChange={(open) => !open && setEditRuleState(null)}
        state={editRuleState}
        onSuccess={refresh}
      />
    </div>
  );
}

interface CreateGroupDialogProps {
  configId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  groupCount: number;
  targets: string[];
  onSuccess: () => void;
}

function CreateGroupDialog({
  configId,
  open,
  onOpenChange,
  groupCount,
  targets,
  onSuccess,
}: CreateGroupDialogProps) {
  const { t } = useTranslation();
  const [name, setName] = useState("");
  const [index, setIndex] = useState<number>(1);
  const [batch, setBatch] = useState<RuleBatchDraft>(() => emptyRuleBatch(targets));
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) return;
    setName("");
    setIndex(Math.max(1, groupCount));
    setBatch(emptyRuleBatch(targets));
    setError("");
  }, [open, groupCount, targets]);

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload = ruleBatchPayload(batch);
    if (!name.trim() || !payload) {
      setError(t("groups.requiredFields"));
      return;
    }
    setError("");
    setSubmitting(true);
    try {
      await createRuleGroup(configId, {
        name: name.trim(),
        index,
        ...payload,
      });
      onSuccess();
      onOpenChange(false);
    } catch (error) {
      setError(t("errors.generic", { message: (error as Error).message }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={onSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>{t("groups.addGroupTitle")}</DialogTitle>
            <DialogDescription>{t("groups.addGroupHint")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="group-name">{t("common.name")}</Label>
            <Input
              id="group-name"
              name="groupName"
              value={name}
              onChange={(event) => {
                setName(event.target.value);
                setError("");
              }}
              autoComplete="off"
              spellCheck={false}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="group-index">{t("groups.indexLabel")}</Label>
            <Input
              id="group-index"
              name="groupIndex"
              type="number"
              inputMode="numeric"
              min={1}
              value={index}
              onChange={(event) => {
                setIndex(Number(event.target.value));
                setError("");
              }}
              autoComplete="off"
              required
              className="max-w-[140px]"
            />
          </div>
          <RuleBatchFields
            idPrefix="create-group"
            draft={batch}
            onChange={(next) => {
              setBatch(next);
              setError("");
            }}
            targets={targets}
          />
          <FormError id="create-group-error" message={error} />
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={submitting} aria-busy={submitting}>
              {submitting ? t("common.loading") : t("common.create")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface RenameGroupDialogProps {
  configId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  currentName: string;
  onSuccess: () => void;
}

function RenameGroupDialog({
  configId,
  open,
  onOpenChange,
  currentName,
  onSuccess,
}: RenameGroupDialogProps) {
  const { t } = useTranslation();
  const [newName, setNewName] = useState(currentName);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (open) {
      setNewName(currentName);
      setError("");
    }
  }, [open, currentName]);

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!newName.trim()) {
      setError(t("groups.nameRequired"));
      return;
    }
    if (newName.trim() === currentName) {
      onOpenChange(false);
      return;
    }
    setError("");
    setSubmitting(true);
    try {
      await updateRuleGroup(configId, currentName, { name: newName.trim() });
      onSuccess();
      onOpenChange(false);
    } catch (error) {
      setError(t("errors.generic", { message: (error as Error).message }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={onSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>
              {t("groups.renameTitle", { name: currentName })}
            </DialogTitle>
            <DialogDescription className="sr-only">
              {t("groups.renameHint")}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="rename-group">{t("common.name")}</Label>
            <Input
              id="rename-group"
              name="groupName"
              value={newName}
              onChange={(event) => {
                setNewName(event.target.value);
                setError("");
              }}
              autoComplete="off"
              spellCheck={false}
              required
            />
          </div>
          <FormError id="rename-group-error" message={error} />
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={submitting} aria-busy={submitting}>
              {submitting ? t("common.loading") : t("common.save")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface AddRuleDialogProps {
  configId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  group: RuleGroup | null;
  targets: string[];
  onSuccess: () => void;
}

function AddRuleDialog({
  configId,
  open,
  onOpenChange,
  group,
  targets,
  onSuccess,
}: AddRuleDialogProps) {
  const { t } = useTranslation();
  const [batch, setBatch] = useState<RuleBatchDraft>(() => emptyRuleBatch(targets));
  const [indexRaw, setIndexRaw] = useState("");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) return;
    setBatch(emptyRuleBatch(targets));
    setIndexRaw("");
    setError("");
  }, [open, targets]);

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const payload = ruleBatchPayload(batch);
    if (!group || !payload) {
      setError(t("groups.rulesRequired"));
      return;
    }
    setError("");
    setSubmitting(true);
    try {
      await addRule(configId, group.name, {
        ...payload,
        index: indexRaw === "" ? undefined : Number(indexRaw),
      });
      onSuccess();
      onOpenChange(false);
    } catch (error) {
      setError(t("errors.generic", { message: (error as Error).message }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={onSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>
              {t("groups.addRuleTitle", { group: group?.name ?? "" })}
            </DialogTitle>
            <DialogDescription className="sr-only">
              {t("groups.addRuleHint")}
            </DialogDescription>
          </DialogHeader>
          <RuleBatchFields
            idPrefix="add-rules"
            draft={batch}
            onChange={(next) => {
              setBatch(next);
              setError("");
            }}
            targets={targets}
          />
          <div className="space-y-2">
            <Label htmlFor="rule-index">
              {t("groups.ruleIndexLabel")}
            </Label>
            <Input
              id="rule-index"
              name="ruleIndex"
              type="number"
              inputMode="numeric"
              min={0}
              value={indexRaw}
              onChange={(event) => {
                setIndexRaw(event.target.value);
                setError("");
              }}
              autoComplete="off"
              className="max-w-[140px]"
            />
          </div>
          <FormError id="add-rules-error" message={error} />
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={submitting} aria-busy={submitting}>
              {submitting ? t("common.loading") : t("groups.addRule")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface RuleBatchFieldsProps {
  idPrefix: string;
  draft: RuleBatchDraft;
  onChange: (draft: RuleBatchDraft) => void;
  targets: string[];
}

function RuleBatchFields({
  idPrefix,
  draft,
  onChange,
  targets,
}: RuleBatchFieldsProps) {
  const { t } = useTranslation();
  const definition = getRuleTypeDefinition(draft.ruleType);
  const usesPolicyTarget = definition.targetMode === "policy";

  return (
    <Tabs
      value={draft.mode}
      onValueChange={(mode) =>
        onChange({ ...draft, mode: mode as RuleBatchDraft["mode"] })
      }
    >
      <TabsList>
        <TabsTrigger value="yaml">{t("groups.batchYaml")}</TabsTrigger>
        <TabsTrigger value="generate">{t("groups.batchGenerate")}</TabsTrigger>
      </TabsList>
      <TabsContent value="yaml" className="space-y-2">
        <Label htmlFor={`${idPrefix}-rules-yaml`}>{t("groups.rulesLabel")}</Label>
        <Textarea
          id={`${idPrefix}-rules-yaml`}
          name="rulesYaml"
          value={draft.yaml}
          onChange={(event) => onChange({ ...draft, yaml: event.target.value })}
          rows={8}
          className="font-mono text-xs"
          placeholder={"- DOMAIN-SUFFIX,example.com,DIRECT\n- IP-CIDR,192.0.2.0/24,DIRECT"}
          autoComplete="off"
          spellCheck={false}
          required={draft.mode === "yaml"}
        />
      </TabsContent>
      <TabsContent value="generate" className="space-y-4">
        <div className="grid gap-3 sm:grid-cols-2">
          <div className="space-y-2">
            <Label htmlFor={`${idPrefix}-rule-type`}>
              {t("groups.ruleTypeLabel")}
            </Label>
            <Select
              value={draft.ruleType}
              onValueChange={(ruleType) =>
                onChange({
                  ...draft,
                  ruleType: ruleType as RuleBatchDraft["ruleType"],
                })
              }
            >
              <SelectTrigger id={`${idPrefix}-rule-type`} name="ruleType">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {RULE_TYPE_GROUPS.map((group) => (
                  <SelectGroup key={group.labelKey}>
                    <SelectLabel>{t(group.labelKey)}</SelectLabel>
                    {group.types.map((ruleType) => (
                      <SelectItem
                        key={ruleType.value}
                        value={ruleType.value}
                        translate="no"
                      >
                        {ruleType.value}
                      </SelectItem>
                    ))}
                  </SelectGroup>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
            {usesPolicyTarget ? (
              <>
                <Label htmlFor={`${idPrefix}-target`}>
                  {t("groups.targetLabel")}
                </Label>
                <Select
                  value={draft.target}
                  onValueChange={(target) => onChange({ ...draft, target })}
                >
                  <SelectTrigger id={`${idPrefix}-target`} name="target">
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {targets.map((target) => (
                      <SelectItem key={target} value={target} translate="no">
                        {target}
                      </SelectItem>
                    ))}
                    <SelectItem value={CUSTOM_TARGET}>
                      {t("groups.customTarget")}
                    </SelectItem>
                  </SelectContent>
                </Select>
              </>
            ) : (
              <>
                <Label htmlFor={`${idPrefix}-sub-rule`}>
                  {t("groups.subRuleLabel")}
                </Label>
                <Input
                  id={`${idPrefix}-sub-rule`}
                  name="subRule"
                  value={draft.subRule}
                  onChange={(event) =>
                    onChange({ ...draft, subRule: event.target.value })
                  }
                  placeholder={t("groups.subRulePlaceholder")}
                  autoComplete="off"
                  spellCheck={false}
                  required={draft.mode === "generate"}
                />
              </>
            )}
          </div>
        </div>
        {usesPolicyTarget && draft.target === CUSTOM_TARGET && (
          <div className="space-y-2">
            <Label htmlFor={`${idPrefix}-custom-target`}>
              {t("groups.customTarget")}
            </Label>
            <Input
              id={`${idPrefix}-custom-target`}
              name="customTarget"
              value={draft.customTarget}
              onChange={(event) =>
                onChange({ ...draft, customTarget: event.target.value })
              }
              autoComplete="off"
              spellCheck={false}
              required
            />
          </div>
        )}
        {definition.valueMode !== "none" && (
          <div className="space-y-2">
            <Label htmlFor={`${idPrefix}-rule-values`}>
              {definition.valueMode === "conditions"
                ? t("groups.conditionsLabel")
                : t("groups.valuesLabel")}
            </Label>
            <Textarea
              id={`${idPrefix}-rule-values`}
              name="ruleValues"
              value={draft.values}
              onChange={(event) =>
                onChange({ ...draft, values: event.target.value })
              }
              rows={7}
              className="font-mono text-xs"
              placeholder={`${definition.placeholder}\n…`}
              autoComplete="off"
              spellCheck={false}
              required={draft.mode === "generate"}
            />
          </div>
        )}
      </TabsContent>
    </Tabs>
  );
}

interface EditRuleDialogProps {
  configId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  state: { group: RuleGroup; index: number; initial: string } | null;
  onSuccess: () => void;
}

function EditRuleDialog({
  configId,
  open,
  onOpenChange,
  state,
  onSuccess,
}: EditRuleDialogProps) {
  const { t } = useTranslation();
  const [rule, setRule] = useState(state?.initial ?? "");
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (open && state) {
      setRule(state.initial);
      setError("");
    }
  }, [open, state]);

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!state || !rule.trim()) {
      setError(t("groups.ruleRequired"));
      return;
    }
    if (rule.trim() === state.initial) {
      onOpenChange(false);
      return;
    }
    setError("");
    setSubmitting(true);
    try {
      await updateRule(configId, state.group.name, state.index, {
        rule: rule.trim(),
      });
      onSuccess();
      onOpenChange(false);
    } catch (error) {
      setError(t("errors.generic", { message: (error as Error).message }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent>
        <form onSubmit={onSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>
              {t("groups.editRuleTitle", {
                group: state?.group.name ?? "",
              })}
            </DialogTitle>
            <DialogDescription className="sr-only">
              {t("groups.editRuleHint")}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="edit-rule-text">{t("groups.ruleLabel")}</Label>
            <Input
              id="edit-rule-text"
              name="rule"
              value={rule}
              onChange={(event) => {
                setRule(event.target.value);
                setError("");
              }}
              autoComplete="off"
              spellCheck={false}
              required
              className="font-mono text-xs"
            />
          </div>
          <FormError id="edit-rule-error" message={error} />
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={submitting} aria-busy={submitting}>
              {submitting ? t("common.loading") : t("common.save")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
