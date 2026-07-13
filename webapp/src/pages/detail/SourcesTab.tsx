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

interface SourcesTabProps {
  id: string;
  rule: MergeRule;
}

export function SourcesTab({ id, rule }: SourcesTabProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [drafts, setDrafts] = useState<GroupDraft[]>(() => ruleToDrafts(rule));
  const [newGroupName, setNewGroupName] = useState("");
  const [expandedSources, setExpandedSources] = useState<Record<string, boolean>>({});

  useEffect(() => {
    setDrafts(ruleToDrafts(rule));
    setExpandedSources({});
  }, [rule]);

  const mutation = useMutation({
    mutationFn: () =>
      updateConfig(id, {
        ...rule,
        configurations: draftsToConfigurations(drafts),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData<MergeRule>(["configs", id], data);
      toast.success(t("detail.sourcesSaved"));
    },
    onError: (error: Error) => {
      toast.error(t("errors.generic", { message: error.message }));
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
    if (!name) return;
    if (drafts.some((group) => group.name === name)) {
      toast.error(t("sources.groupExists"));
      return;
    }
    setDrafts((previous) => [
      ...previous,
      {
        key: nextDraftKey(),
        name,
        includeDirect: false,
        includeGroups: "",
        enableUrlTest: true,
        sources: [],
      },
    ]);
    setNewGroupName("");
  };

  const addSource = (groupIndex: number) => {
    const group = drafts[groupIndex];
    const source = { key: nextDraftKey(), kind: "path" as const, value: "" };
    updateGroup(groupIndex, { ...group, sources: [...group.sources, source] });
    setExpandedSources((previous) => ({ ...previous, [source.key]: true }));
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

  const sourceKindLabel = (kind: SourceKind) => {
    switch (kind) {
      case "path":
        return t("sources.kindPath");
      case "url":
        return t("sources.kindUrl");
      case "cmd":
        return t("sources.kindCmd");
    }
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
        <Input
          value={newGroupName}
          onChange={(event) => setNewGroupName(event.target.value)}
          placeholder={t("sources.newGroupName")}
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
        >
          {mutation.isPending ? t("common.loading") : t("common.save")}
        </Button>
      </div>

      {drafts.map((group, groupIndex) => (
        <section key={group.key} className="border-b pb-5 last:border-b-0">
          <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
            <Input
              value={group.name}
              onChange={(event) =>
                updateGroup(groupIndex, { ...group, name: event.target.value })
              }
              aria-label={t("sources.groupName")}
              className="font-medium sm:max-w-xs"
              required
            />
            <div className="flex items-center gap-1 sm:ml-auto">
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
                  if (window.confirm(t("sources.deleteGroupConfirm", { name: group.name }))) {
                    setDrafts((previous) => previous.filter((_, i) => i !== groupIndex));
                  }
                }}
              >
                <Trash2 className="h-4 w-4" aria-hidden />
              </Button>
            </div>
          </div>

          <Tabs defaultValue="sources" className="mt-3">
            <TabsList>
              <TabsTrigger value="sources">{t("sources.sourcesTab")}</TabsTrigger>
              <TabsTrigger value="options">{t("sources.optionsTab")}</TabsTrigger>
            </TabsList>
            <TabsContent value="sources" className="mt-3 space-y-2">
              <div className="flex justify-end">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon"
                  title={t("sources.addSource")}
                  aria-label={t("sources.addSource")}
                  onClick={() => addSource(groupIndex)}
                >
                  <Plus className="h-4 w-4" aria-hidden />
                </Button>
              </div>
              {group.sources.map((source, sourceIndex) => {
                const isOpen = expandedSources[source.key] ?? false;
                return (
                  <div key={source.key} className="rounded-md border">
                    <div className="flex min-h-10 items-center gap-2 px-2">
                      <button
                        type="button"
                        onClick={() =>
                          setExpandedSources((previous) => ({
                            ...previous,
                            [source.key]: !isOpen,
                          }))
                        }
                        className="flex min-w-0 flex-1 items-center gap-2 text-left"
                      >
                        <ChevronRight
                          className={cn("h-4 w-4 shrink-0 transition-transform", isOpen && "rotate-90")}
                          aria-hidden
                        />
                        <span className="shrink-0 text-xs font-medium">
                          {sourceKindLabel(source.kind)}
                        </span>
                        <span className="truncate text-xs text-muted-foreground">
                          {source.value || sourcePlaceholder(source.kind)}
                        </span>
                      </button>
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon"
                        title={t("sources.moveUp")}
                        aria-label={t("sources.moveUp")}
                        onClick={() => moveSource(groupIndex, sourceIndex, -1)}
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
                        onClick={() => moveSource(groupIndex, sourceIndex, 1)}
                        disabled={sourceIndex === group.sources.length - 1}
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
                            sources: group.sources.filter((_, i) => i !== sourceIndex),
                          })
                        }
                      >
                        <Trash2 className="h-3.5 w-3.5" aria-hidden />
                      </Button>
                    </div>
                    <CollapsiblePane open={isOpen}>
                      <div className="grid gap-3 border-t p-3 sm:grid-cols-[11rem_minmax(0,1fr)]">
                        <div className="space-y-1">
                          <Label>{t("sources.kindLabel")}</Label>
                          <Select
                            value={source.kind}
                            onValueChange={(value) =>
                              updateSource(groupIndex, sourceIndex, {
                                ...source,
                                kind: value as SourceKind,
                              })
                            }
                          >
                            <SelectTrigger>
                              <SelectValue />
                            </SelectTrigger>
                            <SelectContent>
                              <SelectItem value="path">{t("sources.kindPath")}</SelectItem>
                              <SelectItem value="url">{t("sources.kindUrl")}</SelectItem>
                              <SelectItem value="cmd">{t("sources.kindCmd")}</SelectItem>
                            </SelectContent>
                          </Select>
                        </div>
                        <div className="space-y-1">
                          <Label>{t("sources.valueLabel")}</Label>
                          <Input
                            value={source.value}
                            onChange={(event) =>
                              updateSource(groupIndex, sourceIndex, {
                                ...source,
                                value: event.target.value,
                              })
                            }
                            placeholder={sourcePlaceholder(source.kind)}
                            required
                          />
                        </div>
                      </div>
                    </CollapsiblePane>
                  </div>
                );
              })}
            </TabsContent>
            <TabsContent value="options" className="mt-3">
              <div className="grid gap-4 sm:grid-cols-2">
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={group.includeDirect}
                    onChange={(event) =>
                      updateGroup(groupIndex, { ...group, includeDirect: event.target.checked })
                    }
                    className="h-4 w-4 shrink-0 rounded border-input"
                  />
                  <span>{t("sources.includeDirect")}</span>
                </label>
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={group.enableUrlTest}
                    onChange={(event) =>
                      updateGroup(groupIndex, { ...group, enableUrlTest: event.target.checked })
                    }
                    className="h-4 w-4 shrink-0 rounded border-input"
                  />
                  <span>{t("sources.enableUrlTest")}</span>
                </label>
                <div className="space-y-1 sm:col-span-2">
                  <Label>{t("sources.includeGroups")}</Label>
                  <Input
                    value={group.includeGroups}
                    onChange={(event) =>
                      updateGroup(groupIndex, { ...group, includeGroups: event.target.value })
                    }
                    placeholder={t("sources.includeGroupsPlaceholder")}
                  />
                </div>
              </div>
            </TabsContent>
          </Tabs>
        </section>
      ))}
    </div>
  );
}
