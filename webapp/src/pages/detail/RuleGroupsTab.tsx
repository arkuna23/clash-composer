import { FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ChevronDown, ChevronRight, Pencil, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import {
  addRule,
  createRuleGroup,
  deleteRule,
  deleteRuleGroup,
  listRuleGroups,
  updateRule,
  updateRuleGroup,
} from "@/api/ruleGroups";
import type { RuleGroup } from "@/api/types";

const DEFAULT_GROUP = "default";

interface RuleGroupsTabProps {
  id: string;
}

export function RuleGroupsTab({ id }: RuleGroupsTabProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const listKey = ["configs", id, "rule-groups"];
  const [expanded, setExpanded] = useState<Record<string, boolean>>({
    [DEFAULT_GROUP]: true,
  });
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
    setExpanded((prev) => ({ ...prev, [name]: !(prev[name] ?? false) }));
  };

  const groups = groupsQuery.data ?? [];

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
        <p className="text-sm text-muted-foreground">{t("common.loading")}</p>
      ) : groups.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("groups.empty")}</p>
      ) : (
        <div className="space-y-3">
          {groups.map((group) => {
            const isOpen = expanded[group.name] ?? false;
            const isDefault = group.name === DEFAULT_GROUP;
            return (
              <div
                key={group.name}
                className="rounded-md border bg-card"
              >
                <div className="flex items-center gap-2 px-3 py-2">
                  <button
                    type="button"
                    onClick={() => toggle(group.name)}
                    className="flex items-center gap-2 text-sm font-medium hover:text-foreground/80 cursor-pointer"
                  >
                    {isOpen ? (
                      <ChevronDown className="h-4 w-4" aria-hidden />
                    ) : (
                      <ChevronRight className="h-4 w-4" aria-hidden />
                    )}
                    <span>{group.name}</span>
                    {isDefault && (
                      <Badge variant="secondary" className="text-[10px]">
                        default
                      </Badge>
                    )}
                  </button>
                  <span className="text-xs text-muted-foreground ml-2">
                    {t("groups.ruleCount", { count: group.rules.length })}
                  </span>
                  <div className="ml-auto flex gap-1">
                    <Button
                      variant="ghost"
                      size="sm"
                      onClick={() => setAddRuleGroup(group)}
                      className="gap-1.5"
                    >
                      <Plus className="h-4 w-4" aria-hidden />
                      {t("groups.addRule")}
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
                {isOpen && (
                  <div className="border-t px-3 py-2 space-y-1">
                    {group.rules.length === 0 ? (
                      <p className="text-xs text-muted-foreground">
                        {t("groups.empty")}
                      </p>
                    ) : (
                      group.rules.map((rule, index) => (
                        <div
                          key={`${group.name}-${index}`}
                          className="flex items-center gap-2 rounded-md px-2 py-1.5 hover:bg-muted/50"
                        >
                          <span className="text-xs text-muted-foreground w-8 shrink-0 font-mono">
                            {index}
                          </span>
                          <code className="text-xs flex-1 break-all">
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
                )}
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
  onSuccess: () => void;
}

function CreateGroupDialog({
  configId,
  open,
  onOpenChange,
  groupCount,
  onSuccess,
}: CreateGroupDialogProps) {
  const { t } = useTranslation();
  const [name, setName] = useState("");
  const [index, setIndex] = useState<number>(1);
  const [rulesText, setRulesText] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) return;
    setName("");
    setIndex(Math.max(1, groupCount));
    setRulesText("");
  }, [open, groupCount]);

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const rules = rulesText
      .split("\n")
      .map((line) => line.trim())
      .filter(Boolean);
    if (!name.trim() || rules.length === 0) return;
    setSubmitting(true);
    try {
      await createRuleGroup(configId, {
        name: name.trim(),
        index,
        rules,
      });
      onSuccess();
      onOpenChange(false);
    } catch (error) {
      toast.error(t("errors.generic", { message: (error as Error).message }));
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
              value={name}
              onChange={(event) => setName(event.target.value)}
              required
              autoFocus
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="group-index">{t("groups.indexLabel")}</Label>
            <Input
              id="group-index"
              type="number"
              min={1}
              value={index}
              onChange={(event) => setIndex(Number(event.target.value))}
              required
              className="max-w-[140px]"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="group-rules">{t("groups.rulesLabel")}</Label>
            <Textarea
              id="group-rules"
              value={rulesText}
              onChange={(event) => setRulesText(event.target.value)}
              rows={6}
              className="font-mono text-xs"
              spellCheck={false}
              required
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={submitting}>
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

  useEffect(() => {
    if (open) setNewName(currentName);
  }, [open, currentName]);

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!newName.trim() || newName.trim() === currentName) {
      onOpenChange(false);
      return;
    }
    setSubmitting(true);
    try {
      await updateRuleGroup(configId, currentName, { name: newName.trim() });
      onSuccess();
      onOpenChange(false);
    } catch (error) {
      toast.error(t("errors.generic", { message: (error as Error).message }));
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
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="rename-group">{t("common.name")}</Label>
            <Input
              id="rename-group"
              value={newName}
              onChange={(event) => setNewName(event.target.value)}
              required
              autoFocus
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={submitting}>
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
  onSuccess: () => void;
}

function AddRuleDialog({
  configId,
  open,
  onOpenChange,
  group,
  onSuccess,
}: AddRuleDialogProps) {
  const { t } = useTranslation();
  const [rule, setRule] = useState("");
  const [indexRaw, setIndexRaw] = useState("");
  const [submitting, setSubmitting] = useState(false);

  useEffect(() => {
    if (!open) return;
    setRule("");
    setIndexRaw("");
  }, [open]);

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!group || !rule.trim()) return;
    setSubmitting(true);
    try {
      await addRule(configId, group.name, {
        rule: rule.trim(),
        index: indexRaw === "" ? undefined : Number(indexRaw),
      });
      onSuccess();
      onOpenChange(false);
    } catch (error) {
      toast.error(t("errors.generic", { message: (error as Error).message }));
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
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="rule-text">{t("groups.ruleLabel")}</Label>
            <Input
              id="rule-text"
              value={rule}
              onChange={(event) => setRule(event.target.value)}
              required
              autoFocus
              className="font-mono text-xs"
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="rule-index">
              {t("groups.ruleIndexLabel")}
            </Label>
            <Input
              id="rule-index"
              type="number"
              min={0}
              value={indexRaw}
              onChange={(event) => setIndexRaw(event.target.value)}
              className="max-w-[140px]"
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? t("common.loading") : t("common.add")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
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

  useEffect(() => {
    if (open && state) setRule(state.initial);
  }, [open, state]);

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!state || !rule.trim() || rule.trim() === state.initial) {
      onOpenChange(false);
      return;
    }
    setSubmitting(true);
    try {
      await updateRule(configId, state.group.name, state.index, {
        rule: rule.trim(),
      });
      onSuccess();
      onOpenChange(false);
    } catch (error) {
      toast.error(t("errors.generic", { message: (error as Error).message }));
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
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="edit-rule-text">{t("groups.ruleLabel")}</Label>
            <Input
              id="edit-rule-text"
              value={rule}
              onChange={(event) => setRule(event.target.value)}
              required
              autoFocus
              className="font-mono text-xs"
            />
          </div>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={submitting}>
              {submitting ? t("common.loading") : t("common.save")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}
