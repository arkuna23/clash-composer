import { FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useSearchParams } from "react-router-dom";
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
  SelectItem,
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
const RULE_TYPES = [
  "DOMAIN",
  "DOMAIN-SUFFIX",
  "DOMAIN-KEYWORD",
  "IP-CIDR",
  "IP-CIDR6",
] as const;

interface RuleBatchDraft {
  mode: "yaml" | "generate";
  yaml: string;
  ruleType: (typeof RULE_TYPES)[number];
  values: string;
  target: string;
  customTarget: string;
}

function emptyRuleBatch(targets: string[]): RuleBatchDraft {
  return {
    mode: "yaml",
    yaml: "",
    ruleType: "DOMAIN-SUFFIX",
    values: "",
    target: targets[0] ?? CUSTOM_TARGET,
    customTarget: "",
  };
}

function ruleBatchPayload(draft: RuleBatchDraft): AddRulePayload | null {
  if (draft.mode === "yaml") {
    return draft.yaml.trim() ? { rulesYaml: draft.yaml } : null;
  }
  const target =
    draft.target === CUSTOM_TARGET ? draft.customTarget.trim() : draft.target;
  if (!target) return null;
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
  const [searchParams, setSearchParams] = useSearchParams();
  const listKey = ["configs", id, "rule-groups"];
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
    setSearchParams((previous) => {
      const next = new URLSearchParams(previous);
      const expanded = new Set(next.getAll("expanded"));
      if (expanded.has(name)) {
        expanded.delete(name);
      } else {
        expanded.add(name);
      }
      next.delete("expanded");
      for (const groupName of expanded) next.append("expanded", groupName);
      return next;
    }, { replace: true });
  };

  const groups = groupsQuery.data ?? [];
  const expanded = new Set(searchParams.getAll("expanded"));
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
            const isOpen = expanded.has(group.name);
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
          placeholder={"rules:\n  - DOMAIN-SUFFIX,example.com,DIRECT\n  - …"}
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
                {RULE_TYPES.map((ruleType) => (
                  <SelectItem
                    key={ruleType}
                    value={ruleType}
                    translate="no"
                  >
                    {ruleType}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>
          <div className="space-y-2">
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
                <SelectItem value={CUSTOM_TARGET}>{t("groups.customTarget")}</SelectItem>
              </SelectContent>
            </Select>
          </div>
        </div>
        {draft.target === CUSTOM_TARGET && (
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
        <div className="space-y-2">
          <Label htmlFor={`${idPrefix}-rule-values`}>
            {t("groups.valuesLabel")}
          </Label>
          <Textarea
            id={`${idPrefix}-rule-values`}
            name="ruleValues"
            value={draft.values}
            onChange={(event) => onChange({ ...draft, values: event.target.value })}
            rows={7}
            className="font-mono text-xs"
            placeholder={"example.com\nexample.org\n…"}
            autoComplete="off"
            spellCheck={false}
            required={draft.mode === "generate"}
          />
        </div>
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
