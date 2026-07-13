import { FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Pencil, Plus, Trash2 } from "lucide-react";
import { toast } from "sonner";
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
import { FormError } from "@/components/ui/form-error";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Textarea } from "@/components/ui/textarea";
import {
  createRuleProvider,
  deleteRuleProvider,
  listRuleProviders,
  updateRuleProvider,
} from "@/api/ruleProviders";
import type { RuleProvider } from "@/api/types";

interface RuleProvidersTabProps {
  id: string;
}

export function RuleProvidersTab({ id }: RuleProvidersTabProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const listKey = ["configs", id, "rule-providers"];
  const [editing, setEditing] = useState<{
    name: string;
    body: RuleProvider;
  } | null>(null);
  const [creating, setCreating] = useState(false);

  const listQuery = useQuery({
    queryKey: listKey,
    queryFn: ({ signal }) => listRuleProviders(id, signal),
  });

  const deleteMutation = useMutation({
    mutationFn: (name: string) => deleteRuleProvider(id, name),
    onSuccess: (_data, name) => {
      toast.success(t("providers.deleted", { name }));
      void queryClient.invalidateQueries({ queryKey: listKey });
    },
    onError: (error: Error) => {
      toast.error(t("errors.generic", { message: error.message }));
    },
  });

  const onDelete = (name: string) => {
    if (!window.confirm(t("providers.deleteConfirm", { name }))) return;
    deleteMutation.mutate(name);
  };

  const providers = Object.entries(listQuery.data ?? {});

  return (
    <div className="space-y-4">
      <div className="flex justify-end">
        <Button onClick={() => setCreating(true)} className="gap-1.5">
          <Plus className="h-4 w-4" aria-hidden />
          {t("providers.add")}
        </Button>
      </div>
      {listQuery.isLoading ? (
        <p
          role="status"
          aria-live="polite"
          className="text-sm text-muted-foreground"
        >
          {t("common.loading")}
        </p>
      ) : listQuery.isError ? (
        <p role="alert" className="text-sm text-destructive">
          {(listQuery.error as Error).message}
        </p>
      ) : providers.length === 0 ? (
        <p className="text-sm text-muted-foreground">{t("providers.empty")}</p>
      ) : (
        <Table>
          <TableHeader>
            <TableRow>
              <TableHead>{t("common.name")}</TableHead>
              <TableHead>type</TableHead>
              <TableHead>behavior</TableHead>
              <TableHead>url / path</TableHead>
              <TableHead className="text-right pr-3">
                {t("common.actions")}
              </TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {providers.map(([name, body]) => {
              const record = body as Record<string, unknown>;
              return (
                <TableRow key={name}>
                  <TableCell className="font-medium" translate="no">
                    {name}
                  </TableCell>
                  <TableCell translate="no">{String(record.type ?? "")}</TableCell>
                  <TableCell translate="no">
                    {String(record.behavior ?? "")}
                  </TableCell>
                  <TableCell className="max-w-[280px] truncate">
                    <span
                      className="text-xs text-muted-foreground"
                      translate="no"
                    >
                      {String(record.url ?? record.path ?? "")}
                    </span>
                  </TableCell>
                  <TableCell className="text-right pr-3">
                    <div className="flex justify-end gap-1">
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={t("common.edit")}
                        onClick={() =>
                          setEditing({ name, body: body as RuleProvider })
                        }
                      >
                        <Pencil className="h-4 w-4" aria-hidden />
                      </Button>
                      <Button
                        variant="ghost"
                        size="icon"
                        aria-label={t("common.delete")}
                        onClick={() => onDelete(name)}
                      >
                        <Trash2 className="h-4 w-4" aria-hidden />
                      </Button>
                    </div>
                  </TableCell>
                </TableRow>
              );
            })}
          </TableBody>
        </Table>
      )}

      <ProviderDialog
        configId={id}
        open={creating}
        onOpenChange={setCreating}
        mode="create"
        onSuccess={() => queryClient.invalidateQueries({ queryKey: listKey })}
      />
      <ProviderDialog
        configId={id}
        open={editing !== null}
        onOpenChange={(open) => !open && setEditing(null)}
        mode="edit"
        initialName={editing?.name}
        initialBody={editing?.body}
        onSuccess={() => queryClient.invalidateQueries({ queryKey: listKey })}
      />
    </div>
  );
}

interface ProviderDialogProps {
  configId: string;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  mode: "create" | "edit";
  initialName?: string;
  initialBody?: RuleProvider;
  onSuccess: () => void;
}

const SAMPLE_BODY = JSON.stringify(
  {
    type: "http",
    behavior: "domain",
    url: "https://example.com/google.txt",
    path: "./ruleset/google.yaml",
    interval: 86400,
  },
  null,
  2,
);

function ProviderDialog({
  configId,
  open,
  onOpenChange,
  mode,
  initialName,
  initialBody,
  onSuccess,
}: ProviderDialogProps) {
  const { t } = useTranslation();
  const [name, setName] = useState(initialName ?? "");
  const [bodyText, setBodyText] = useState(
    initialBody ? JSON.stringify(initialBody, null, 2) : SAMPLE_BODY,
  );
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");

  useEffect(() => {
    if (!open) return;
    setName(initialName ?? "");
    setBodyText(
      initialBody ? JSON.stringify(initialBody, null, 2) : SAMPLE_BODY,
    );
    setError("");
  }, [open, initialName, initialBody]);

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!name.trim()) {
      setError(t("providers.nameRequired"));
      return;
    }
    setError("");
    let parsed: RuleProvider;
    try {
      parsed = JSON.parse(bodyText) as RuleProvider;
      if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) {
        throw new Error(t("providers.invalidJson"));
      }
    } catch (error) {
      setError(error instanceof Error ? error.message : t("providers.invalidJson"));
      return;
    }
    setSubmitting(true);
    try {
      if (mode === "create") {
        await createRuleProvider(configId, name.trim(), parsed);
      } else {
        await updateRuleProvider(configId, name.trim(), parsed);
      }
      toast.success(t("providers.saved"));
      setError("");
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
      <DialogContent className="max-w-2xl">
        <form onSubmit={onSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>
              {mode === "create" ? t("providers.add") : t("providers.edit")}
            </DialogTitle>
            <DialogDescription>{t("providers.bodyHint")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="provider-name">{t("providers.nameLabel")}</Label>
            <Input
              id="provider-name"
              name="providerName"
              value={name}
              onChange={(event) => {
                setName(event.target.value);
                setError("");
              }}
              placeholder={t("providers.namePlaceholder")}
              autoComplete="off"
              spellCheck={false}
              disabled={mode === "edit"}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="provider-body">{t("providers.bodyLabel")}</Label>
            <Textarea
              id="provider-body"
              name="providerBody"
              value={bodyText}
              onChange={(event) => {
                setBodyText(event.target.value);
                setError("");
              }}
              rows={12}
              className="font-mono text-xs"
              autoComplete="off"
              spellCheck={false}
              required
            />
          </div>
          <FormError id="provider-error" message={error} />
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
