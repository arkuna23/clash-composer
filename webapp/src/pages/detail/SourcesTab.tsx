import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { ArrowDown, ArrowUp, Plus, Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { updateConfig } from "@/api/configs";
import type { ConfigGroup, ConfigSource, MergeRule } from "@/api/types";

type SourceKind = "path" | "url" | "cmd";

interface SourceDraft {
  kind: SourceKind;
  value: string;
}

interface GroupDraft {
  name: string;
  includeDirect: boolean;
  includeGroups: string;
  enableUrlTest: boolean;
  sources: SourceDraft[];
}

function sourceToDraft(source: ConfigSource): SourceDraft {
  if (source.path !== undefined && source.path !== "") {
    return { kind: "path", value: source.path };
  }
  if (source.url !== undefined && source.url !== "") {
    return { kind: "url", value: source.url };
  }
  if (source.cmd !== undefined && source.cmd !== "") {
    return { kind: "cmd", value: source.cmd };
  }
  return { kind: "path", value: "" };
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

function normalizeConfigGroup(group: ConfigGroup | ConfigSource[]): ConfigGroup {
  if (Array.isArray(group)) {
    return { sources: group };
  }
  return group;
}

function ruleToDrafts(rule: MergeRule): GroupDraft[] {
  const groups = rule.configurations ?? {};
  return Object.keys(groups).map((name) => {
    const group = normalizeConfigGroup(groups[name]);
    return {
      name,
      includeDirect: group.includeDirect ?? false,
      includeGroups: (group.includeGroups ?? []).join(", "),
      enableUrlTest: group.enableUrlTest ?? true,
      sources: (group.sources ?? []).map(sourceToDraft),
    };
  });
}

function parseIncludeGroups(value: string): string[] {
  return value
    .split(",")
    .map((name) => name.trim())
    .filter(Boolean);
}

function draftsToConfigurations(drafts: GroupDraft[]): Record<string, ConfigGroup> {
  const result: Record<string, ConfigGroup> = {};
  for (const draft of drafts) {
    if (!draft.name.trim()) {
      continue;
    }
    result[draft.name.trim()] = {
      sources: draft.sources.map(draftToSource),
      includeDirect: draft.includeDirect,
      includeGroups: parseIncludeGroups(draft.includeGroups),
      enableUrlTest: draft.enableUrlTest,
    };
  }
  return result;
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

  // Reset local drafts when the upstream rule changes (e.g. after invalidation).
  useEffect(() => {
    setDrafts(ruleToDrafts(rule));
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
    setDrafts((prev) => prev.map((group, i) => (i === index ? next : group)));
  };

  const removeGroup = (index: number) => {
    setDrafts((prev) => prev.filter((_, i) => i !== index));
  };

  const renameGroup = (index: number, name: string) => {
    updateGroup(index, { ...drafts[index], name });
  };

  const addGroup = () => {
    const name = newGroupName.trim();
    if (!name) return;
    if (drafts.some((g) => g.name === name)) {
      toast.error(t("sources.groupExists"));
      return;
    }
    setDrafts((prev) => [
      ...prev,
      {
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
    const next = { ...drafts[groupIndex] };
    next.sources = [...next.sources, { kind: "path", value: "" }];
    updateGroup(groupIndex, next);
  };

  const updateSource = (
    groupIndex: number,
    sourceIndex: number,
    next: SourceDraft,
  ) => {
    const group = { ...drafts[groupIndex] };
    group.sources = group.sources.map((source, i) =>
      i === sourceIndex ? next : source,
    );
    updateGroup(groupIndex, group);
  };

  const removeSource = (groupIndex: number, sourceIndex: number) => {
    const group = { ...drafts[groupIndex] };
    group.sources = group.sources.filter((_, i) => i !== sourceIndex);
    updateGroup(groupIndex, group);
  };

  const moveSource = (
    groupIndex: number,
    sourceIndex: number,
    direction: -1 | 1,
  ) => {
    const group = { ...drafts[groupIndex] };
    const target = sourceIndex + direction;
    if (target < 0 || target >= group.sources.length) return;
    const next = [...group.sources];
    [next[sourceIndex], next[target]] = [next[target], next[sourceIndex]];
    group.sources = next;
    updateGroup(groupIndex, group);
  };

  const updateIncludeDirect = (groupIndex: number, includeDirect: boolean) => {
    updateGroup(groupIndex, { ...drafts[groupIndex], includeDirect });
  };

  const updateIncludeGroups = (groupIndex: number, includeGroups: string) => {
    updateGroup(groupIndex, { ...drafts[groupIndex], includeGroups });
  };

  const updateEnableUrlTest = (groupIndex: number, enableUrlTest: boolean) => {
    updateGroup(groupIndex, { ...drafts[groupIndex], enableUrlTest });
  };

  const valuePlaceholder = (kind: SourceKind) => {
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
    <div className="space-y-6">
      <div className="space-y-4">
        {drafts.length === 0 ? (
          <p className="text-sm text-muted-foreground">
            {t("configs.empty")}
          </p>
        ) : (
          drafts.map((group, groupIndex) => (
            <div
              key={groupIndex}
              className="rounded-md border bg-card p-4 space-y-3"
            >
              <div className="flex items-center gap-2">
                <Label className="shrink-0">{t("sources.groupName")}</Label>
                <Input
                  value={group.name}
                  onChange={(event) =>
                    renameGroup(groupIndex, event.target.value)
                  }
                  className="max-w-xs"
                  required
                />
                <div className="ml-auto flex gap-2">
                  <Button
                    type="button"
                    variant="ghost"
                    size="sm"
                    onClick={() => addSource(groupIndex)}
                    className="gap-1.5"
                  >
                    <Plus className="h-4 w-4" aria-hidden />
                    {t("sources.addSource")}
                  </Button>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon"
                    onClick={() => {
                      if (
                        window.confirm(
                          t("sources.deleteGroupConfirm", { name: group.name }),
                        )
                      ) {
                        removeGroup(groupIndex);
                      }
                    }}
                    aria-label={t("sources.deleteGroup")}
                  >
                    <Trash2 className="h-4 w-4" aria-hidden />
                  </Button>
                </div>
              </div>
              <div className="grid gap-3 border-t pt-3 sm:grid-cols-2 sm:items-center">
                <label className="flex items-center gap-2 text-sm">
                  <input
                    type="checkbox"
                    checked={group.includeDirect}
                    onChange={(event) =>
                      updateIncludeDirect(groupIndex, event.target.checked)
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
                      updateEnableUrlTest(groupIndex, event.target.checked)
                    }
                    className="h-4 w-4 shrink-0 rounded border-input"
                  />
                  <span>{t("sources.enableUrlTest")}</span>
                </label>
                <div className="space-y-1 sm:col-span-2">
                  <Label htmlFor={`include-groups-${groupIndex}`}>
                    {t("sources.includeGroups")}
                  </Label>
                  <Input
                    id={`include-groups-${groupIndex}`}
                    value={group.includeGroups}
                    onChange={(event) =>
                      updateIncludeGroups(groupIndex, event.target.value)
                    }
                    placeholder={t("sources.includeGroupsPlaceholder")}
                  />
                  <p className="text-xs text-muted-foreground">
                    {t("sources.includeGroupsHelp")}
                  </p>
                </div>
              </div>
              {group.sources.length === 0 ? (
                <p className="text-xs text-muted-foreground">
                  {t("sources.addSource")}…
                </p>
              ) : (
                <div className="space-y-2">
                  {group.sources.map((source, sourceIndex) => (
                    <div
                      key={sourceIndex}
                      className="flex flex-col gap-2 rounded-md border p-2 sm:flex-row sm:items-center"
                    >
                      <Select
                        value={source.kind}
                        onValueChange={(value) =>
                          updateSource(groupIndex, sourceIndex, {
                            ...source,
                            kind: value as SourceKind,
                          })
                        }
                      >
                        <SelectTrigger className="w-full sm:w-44">
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
                      <Input
                        value={source.value}
                        onChange={(event) =>
                          updateSource(groupIndex, sourceIndex, {
                            ...source,
                            value: event.target.value,
                          })
                        }
                        placeholder={valuePlaceholder(source.kind)}
                        required
                      />
                      <div className="flex gap-1">
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          aria-label={t("sources.moveUp")}
                          onClick={() =>
                            moveSource(groupIndex, sourceIndex, -1)
                          }
                          disabled={sourceIndex === 0}
                        >
                          <ArrowUp className="h-4 w-4" aria-hidden />
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          aria-label={t("sources.moveDown")}
                          onClick={() =>
                            moveSource(groupIndex, sourceIndex, 1)
                          }
                          disabled={sourceIndex === group.sources.length - 1}
                        >
                          <ArrowDown className="h-4 w-4" aria-hidden />
                        </Button>
                        <Button
                          type="button"
                          variant="ghost"
                          size="icon"
                          aria-label={t("common.delete")}
                          onClick={() => removeSource(groupIndex, sourceIndex)}
                        >
                          <Trash2 className="h-4 w-4" aria-hidden />
                        </Button>
                      </div>
                    </div>
                  ))}
                </div>
              )}
            </div>
          ))
        )}
      </div>
      <div className="flex flex-col gap-2 sm:flex-row sm:items-center">
        <Input
          value={newGroupName}
          onChange={(event) => setNewGroupName(event.target.value)}
          placeholder={t("sources.newGroupName")}
          className="max-w-xs"
        />
        <Button type="button" variant="outline" onClick={addGroup}>
          {t("sources.addGroup")}
        </Button>
      </div>
      <Button
        type="button"
        onClick={() => mutation.mutate()}
        disabled={mutation.isPending}
      >
        {mutation.isPending ? t("common.loading") : t("common.save")}
      </Button>
    </div>
  );
}
