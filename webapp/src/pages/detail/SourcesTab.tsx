import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowDown, ArrowUp, ChevronRight, Plus, Trash2 } from "lucide-react";
import { cn } from "@/lib/utils";
import { Button } from "@/components/ui/button";
import { CollapsiblePane } from "@/components/ui/collapsible-pane";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { FormError } from "@/components/ui/form-error";
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
import { updateConfig } from "@/api/configs";
import type { ConfigGroup, ConfigSource, MergeRule } from "@/api/types";

type SourceKind = "path" | "url" | "cmd";

let draftKey = 0;
const nextDraftKey = () => `draft-${draftKey++}`;

interface SourceDraft {
  key: string;
  kind: SourceKind;
  value: string;
}

interface GroupDraft {
  key: string;
  name: string;
  includeDirect: boolean;
  includeGroups: string;
  enableUrlTest: boolean;
  sources: SourceDraft[];
}

function sourceToDraft(source: ConfigSource): SourceDraft {
  if (source.path) return { key: nextDraftKey(), kind: "path", value: source.path };
  if (source.url) return { key: nextDraftKey(), kind: "url", value: source.url };
  if (source.cmd) return { key: nextDraftKey(), kind: "cmd", value: source.cmd };
  return { key: nextDraftKey(), kind: "path", value: "" };
}

function draftToSource(draft: SourceDraft): ConfigSource {
  switch (draft.kind) {
    case "path":
      return { path: draft.value };
    case "url":
      return { url: draft.value };
    case "cmd":
      return { cmd: draft.value };
  }
}

function ruleToDrafts(rule: MergeRule): GroupDraft[] {
  return (rule.configurations ?? []).map((group) => ({
    key: nextDraftKey(),
    name: group.name,
    includeDirect: group.includeDirect ?? false,
    includeGroups: (group.includeGroups ?? []).join(", "),
    enableUrlTest: group.enableUrlTest ?? true,
    sources: (group.sources ?? []).map(sourceToDraft),
  }));
}

function parseIncludeGroups(value: string): string[] {
  return value
    .split(",")
    .map((name) => name.trim())
    .filter(Boolean);
}

function draftsToConfigurations(drafts: GroupDraft[]): ConfigGroup[] {
  return drafts
    .filter((draft) => draft.name.trim())
    .map((draft) => ({
      name: draft.name.trim(),
      sources: draft.sources.map(draftToSource),
      includeDirect: draft.includeDirect,
      includeGroups: parseIncludeGroups(draft.includeGroups),
      enableUrlTest: draft.enableUrlTest,
    }));
}

function normalizeConfigurations(groups: ConfigGroup[]): ConfigGroup[] {
  return groups.map((group) => ({
    name: group.name,
    sources: group.sources ?? [],
    includeDirect: group.includeDirect ?? false,
    includeGroups: group.includeGroups ?? [],
    enableUrlTest: group.enableUrlTest ?? true,
  }));
}

interface SourcesTabProps {
  id: string;
  rule: MergeRule;
  onDirtyChange?: (dirty: boolean) => void;
}

export function SourcesTab({ id, rule, onDirtyChange }: SourcesTabProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [drafts, setDrafts] = useState<GroupDraft[]>(() => ruleToDrafts(rule));
  const [newGroupName, setNewGroupName] = useState("");
  const [expandedGroups, setExpandedGroups] = useState<Set<string>>(
    () => new Set(),
  );
  const [error, setError] = useState("");
  const isDirty =
    JSON.stringify(draftsToConfigurations(drafts)) !==
    JSON.stringify(normalizeConfigurations(rule.configurations ?? []));

  useEffect(() => {
    onDirtyChange?.(isDirty);
  }, [isDirty, onDirtyChange]);

  useEffect(
    () => () => {
      onDirtyChange?.(false);
    },
    [onDirtyChange],
  );

  useEffect(() => {
    setDrafts(ruleToDrafts(rule));
    setExpandedGroups(new Set());
  }, [rule]);

  const mutation = useMutation({
    mutationFn: () =>
      updateConfig(id, {
        ...rule,
        configurations: draftsToConfigurations(drafts),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData<MergeRule>(["configs", id], data);
      setError("");
      toast.success(t("detail.sourcesSaved"));
    },
    onError: (error: Error) => {
      setError(t("errors.generic", { message: error.message }));
    },
  });

  const updateGroup = (index: number, next: GroupDraft) => {
    setDrafts((previous) => previous.map((group, i) => (i === index ? next : group)));
  };

  const moveGroup = (index: number, direction: -1 | 1) => {
    const target = index + direction;
    if (target < 0 || target >= drafts.length) return;
    setDrafts((previous) => {
      const next = [...previous];
      [next[index], next[target]] = [next[target], next[index]];
      return next;
    });
  };

  const addGroup = () => {
    const name = newGroupName.trim();
    if (!name) {
      setError(t("sources.groupNameRequired"));
      return;
    }
    if (drafts.some((group) => group.name === name)) {
      setError(t("sources.groupExists"));
      return;
    }
    setError("");
    const key = nextDraftKey();
    setDrafts((previous) => [
      ...previous,
      {
        key,
        name,
        includeDirect: false,
        includeGroups: "",
        enableUrlTest: true,
        sources: [],
      },
    ]);
    setExpandedGroups((previous) => new Set(previous).add(key));
    setNewGroupName("");
  };

  const addSource = (groupIndex: number) => {
    const group = drafts[groupIndex];
    const source = { key: nextDraftKey(), kind: "path" as const, value: "" };
    updateGroup(groupIndex, { ...group, sources: [...group.sources, source] });
  };

  const toggleGroup = (key: string) => {
    setExpandedGroups((previous) => {
      const next = new Set(previous);
      if (next.has(key)) {
        next.delete(key);
      } else {
        next.add(key);
      }
      return next;
    });
  };

  const updateSource = (groupIndex: number, sourceIndex: number, next: SourceDraft) => {
    const group = drafts[groupIndex];
    updateGroup(groupIndex, {
      ...group,
      sources: group.sources.map((source, i) => (i === sourceIndex ? next : source)),
    });
  };

  const moveSource = (groupIndex: number, sourceIndex: number, direction: -1 | 1) => {
    const group = drafts[groupIndex];
    const target = sourceIndex + direction;
    if (target < 0 || target >= group.sources.length) return;
    const sources = [...group.sources];
    [sources[sourceIndex], sources[target]] = [sources[target], sources[sourceIndex]];
    updateGroup(groupIndex, { ...group, sources });
  };

  const sourcePlaceholder = (kind: SourceKind) => {
    switch (kind) {
      case "path":
        return t("sources.valuePath");
      case "url":
        return t("sources.valueUrl");
      case "cmd":
        return t("sources.valueCmd");
    }
  };

  return (
    <div className="space-y-5">
      <div className="flex flex-col gap-2 border-b pb-4 sm:flex-row sm:items-center">
        <Label htmlFor="sources-new-group" className="sr-only">
          {t("sources.newGroupLabel")}
        </Label>
        <Input
          id="sources-new-group"
          name="newGroupName"
          value={newGroupName}
          onChange={(event) => {
            setNewGroupName(event.target.value);
            setError("");
          }}
          placeholder={t("sources.newGroupName")}
          autoComplete="off"
          spellCheck={false}
          aria-describedby={error ? "sources-error" : undefined}
          aria-invalid={!!error}
          className="sm:max-w-xs"
        />
        <Button type="button" variant="outline" onClick={addGroup}>
          <Plus className="mr-1.5 h-4 w-4" aria-hidden />
          {t("sources.addGroup")}
        </Button>
        <Button
          type="button"
          className="sm:ml-auto"
          onClick={() => mutation.mutate()}
          disabled={mutation.isPending}
          aria-busy={mutation.isPending}
        >
          {mutation.isPending ? t("common.loading") : t("detail.saveSources")}
        </Button>
      </div>
      <FormError id="sources-error" message={error} />

      {drafts.map((group, groupIndex) => {
        const isOpen = expandedGroups.has(group.key);
        const contentID = `source-group-${group.key}-details`;
        return (
          <section
            key={group.key}
            className="rounded-md border bg-card text-card-foreground"
          >
            <div className="flex flex-col gap-2 p-2 sm:flex-row sm:items-center">
              <div className="flex min-w-0 flex-1 items-center gap-2">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  className="shrink-0"
                  title={
                    isOpen
                      ? t("sources.collapseGroup")
                      : t("sources.expandGroup")
                  }
                  aria-label={
                    isOpen
                      ? t("sources.collapseGroup")
                      : t("sources.expandGroup")
                  }
                  aria-expanded={isOpen}
                  aria-controls={contentID}
                  onClick={() => toggleGroup(group.key)}
                >
                  <ChevronRight
                    className={cn(
                      "h-4 w-4 transition-transform duration-200 ease-out motion-reduce:transition-none",
                      isOpen && "rotate-90",
                    )}
                    aria-hidden
                  />
                </Button>
                <Input
                  name={`configurations.${groupIndex}.name`}
                  value={group.name}
                  onChange={(event) =>
                    updateGroup(groupIndex, {
                      ...group,
                      name: event.target.value,
                    })
                  }
                  aria-label={t("sources.groupName")}
                  autoComplete="off"
                  spellCheck={false}
                  className="min-w-0 flex-1 font-medium sm:max-w-xs"
                  required
                />
              </div>
              <div className="ml-auto flex items-center gap-1">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  title={t("sources.moveUp")}
                  aria-label={t("sources.moveUp")}
                  onClick={() => moveGroup(groupIndex, -1)}
                  disabled={groupIndex === 0}
                >
                  <ArrowUp className="h-4 w-4" aria-hidden />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  title={t("sources.moveDown")}
                  aria-label={t("sources.moveDown")}
                  onClick={() => moveGroup(groupIndex, 1)}
                  disabled={groupIndex === drafts.length - 1}
                >
                  <ArrowDown className="h-4 w-4" aria-hidden />
                </Button>
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  title={t("sources.deleteGroup")}
                  aria-label={t("sources.deleteGroup")}
                  onClick={() => {
                    if (
                      window.confirm(
                        t("sources.deleteGroupConfirm", { name: group.name }),
                      )
                    ) {
                      setDrafts((previous) =>
                        previous.filter((_, i) => i !== groupIndex),
                      );
                      setExpandedGroups((previous) => {
                        const next = new Set(previous);
                        next.delete(group.key);
                        return next;
                      });
                    }
                  }}
                >
                  <Trash2 className="h-4 w-4" aria-hidden />
                </Button>
              </div>
            </div>

            <CollapsiblePane id={contentID} open={isOpen}>
              <div className="border-t px-3 pb-3">
                <Tabs defaultValue="sources" className="mt-3">
                  <TabsList>
                    <TabsTrigger value="sources">
                      {t("sources.sourcesTab")}
                    </TabsTrigger>
                    <TabsTrigger value="options">
                      {t("sources.optionsTab")}
                    </TabsTrigger>
                  </TabsList>
                  <TabsContent value="sources" className="mt-3 space-y-2">
                    <div className="flex min-h-10 items-start justify-between gap-3">
                      <p className="max-w-2xl py-2 text-sm text-muted-foreground">
                        {t("sources.description")}
                      </p>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        className="shrink-0"
                        title={t("sources.addSource")}
                        aria-label={t("sources.addSource")}
                        onClick={() => addSource(groupIndex)}
                      >
                        <Plus className="h-4 w-4" aria-hidden />
                      </Button>
                    </div>
                    <div className="divide-y">
                      {group.sources.map((source, sourceIndex) => (
                        <div
                          key={source.key}
                          className="grid gap-3 py-3 sm:grid-cols-[11rem_minmax(0,1fr)_auto] sm:items-end"
                        >
                          <div className="space-y-1">
                            <Label htmlFor={`source-${source.key}-kind`}>
                              {t("sources.kindLabel")}
                            </Label>
                            <Select
                              value={source.kind}
                              onValueChange={(value) =>
                                updateSource(groupIndex, sourceIndex, {
                                  ...source,
                                  kind: value as SourceKind,
                                })
                              }
                            >
                              <SelectTrigger
                                id={`source-${source.key}-kind`}
                                name={`configurations.${groupIndex}.sources.${sourceIndex}.kind`}
                              >
                                <SelectValue />
                              </SelectTrigger>
                              <SelectContent>
                                <SelectItem value="path">
                                  {t("sources.kindPath")}
                                </SelectItem>
                                <SelectItem value="url">
                                  {t("sources.kindUrl")}
                                </SelectItem>
                                <SelectItem value="cmd">
                                  {t("sources.kindCmd")}
                                </SelectItem>
                              </SelectContent>
                            </Select>
                          </div>
                          <div className="min-w-0 space-y-1">
                            <Label htmlFor={`source-${source.key}-value`}>
                              {t("sources.valueLabel")}
                            </Label>
                            <Input
                              id={`source-${source.key}-value`}
                              name={`configurations.${groupIndex}.sources.${sourceIndex}.value`}
                              type={source.kind === "url" ? "url" : "text"}
                              inputMode={
                                source.kind === "url" ? "url" : undefined
                              }
                              value={source.value}
                              onChange={(event) =>
                                updateSource(groupIndex, sourceIndex, {
                                  ...source,
                                  value: event.target.value,
                                })
                              }
                              placeholder={sourcePlaceholder(source.kind)}
                              autoComplete="off"
                              spellCheck={false}
                              required
                            />
                          </div>
                          <div className="flex justify-end gap-1">
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              title={t("sources.moveUp")}
                              aria-label={t("sources.moveUp")}
                              onClick={() =>
                                moveSource(groupIndex, sourceIndex, -1)
                              }
                              disabled={sourceIndex === 0}
                            >
                              <ArrowUp className="h-3.5 w-3.5" aria-hidden />
                            </Button>
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              title={t("sources.moveDown")}
                              aria-label={t("sources.moveDown")}
                              onClick={() =>
                                moveSource(groupIndex, sourceIndex, 1)
                              }
                              disabled={
                                sourceIndex === group.sources.length - 1
                              }
                            >
                              <ArrowDown className="h-3.5 w-3.5" aria-hidden />
                            </Button>
                            <Button
                              type="button"
                              variant="ghost"
                              size="icon"
                              title={t("common.delete")}
                              aria-label={t("common.delete")}
                              onClick={() =>
                                updateGroup(groupIndex, {
                                  ...group,
                                  sources: group.sources.filter(
                                    (_, i) => i !== sourceIndex,
                                  ),
                                })
                              }
                            >
                              <Trash2 className="h-3.5 w-3.5" aria-hidden />
                            </Button>
                          </div>
                        </div>
                      ))}
                    </div>
                  </TabsContent>
                  <TabsContent value="options" className="mt-3">
                    <div className="grid gap-4 sm:grid-cols-2">
                      <label className="flex items-center gap-2 text-sm">
                        <input
                          type="checkbox"
                          name={`configurations.${groupIndex}.includeDirect`}
                          checked={group.includeDirect}
                          onChange={(event) =>
                            updateGroup(groupIndex, {
                              ...group,
                              includeDirect: event.target.checked,
                            })
                          }
                          className="h-4 w-4 shrink-0 rounded border-input"
                        />
                        <span>{t("sources.includeDirect")}</span>
                      </label>
                      <label className="flex items-center gap-2 text-sm">
                        <input
                          type="checkbox"
                          name={`configurations.${groupIndex}.enableUrlTest`}
                          checked={group.enableUrlTest}
                          onChange={(event) =>
                            updateGroup(groupIndex, {
                              ...group,
                              enableUrlTest: event.target.checked,
                            })
                          }
                          className="h-4 w-4 shrink-0 rounded border-input"
                        />
                        <span>{t("sources.enableUrlTest")}</span>
                      </label>
                      <div className="space-y-1 sm:col-span-2">
                        <Label htmlFor={`group-${group.key}-include-groups`}>
                          {t("sources.includeGroups")}
                        </Label>
                        <Input
                          id={`group-${group.key}-include-groups`}
                          name={`configurations.${groupIndex}.includeGroups`}
                          value={group.includeGroups}
                          onChange={(event) =>
                            updateGroup(groupIndex, {
                              ...group,
                              includeGroups: event.target.value,
                            })
                          }
                          placeholder={t("sources.includeGroupsPlaceholder")}
                          autoComplete="off"
                          spellCheck={false}
                        />
                      </div>
                    </div>
                  </TabsContent>
                </Tabs>
              </div>
            </CollapsiblePane>
          </section>
        );
      })}
    </div>
  );
}
