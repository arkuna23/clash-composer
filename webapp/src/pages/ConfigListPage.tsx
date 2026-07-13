import { ChangeEvent, FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { Plus, Trash2, FilePlus2, Upload } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { ApiError } from "@/api/client";
import { createConfig, deleteConfig, listConfigs } from "@/api/configs";
import { uploadFile } from "@/api/files";
import type { MergeRule, RulesetStrategy } from "@/api/types";

const RULESET_STRATEGIES: { value: RulesetStrategy; label: string }[] = [
  { value: "", label: "—" },
  { value: "url-ruleset", label: "url-ruleset" },
  { value: "replace-ruleset", label: "replace-ruleset" },
];

export function ConfigListPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const [createOpen, setCreateOpen] = useState(false);
  const [uploadOpen, setUploadOpen] = useState(false);

  const listQuery = useQuery({
    queryKey: ["configs"],
    queryFn: ({ signal }) => listConfigs(signal),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: string) => deleteConfig(id),
    onSuccess: (_data, id) => {
      toast.success(t("configs.deleteSuccess", { id }));
      void queryClient.invalidateQueries({ queryKey: ["configs"] });
    },
    onError: (error: ApiError | Error) => {
      toast.error(t("errors.generic", { message: error.message }));
    },
  });

  const onDelete = (id: string) => {
    if (!window.confirm(t("configs.deleteConfirm", { id }))) {
      return;
    }
    deleteMutation.mutate(id);
  };

  return (
    <div className="space-y-6">
      <Card>
        <CardHeader className="flex flex-col gap-3 space-y-0 sm:flex-row sm:items-center sm:justify-between">
          <div className="space-y-1">
            <CardTitle className="text-xl">{t("configs.title")}</CardTitle>
            <CardDescription>{t("app.subtitle")}</CardDescription>
          </div>
          <div className="flex w-full flex-col gap-2 sm:w-auto sm:flex-row">
            <Button
              variant="outline"
              onClick={() => setUploadOpen(true)}
              className="gap-1.5"
            >
              <Upload className="h-4 w-4" aria-hidden />
              {t("files.upload")}
            </Button>
            <Button onClick={() => setCreateOpen(true)} className="gap-1.5">
              <Plus className="h-4 w-4" aria-hidden />
              {t("configs.create")}
            </Button>
          </div>
        </CardHeader>
        <CardContent>
          {listQuery.isLoading ? (
            <p className="text-sm text-muted-foreground">
              {t("common.loading")}
            </p>
          ) : listQuery.isError ? (
            <p className="text-sm text-destructive">
              {(listQuery.error as Error).message}
            </p>
          ) : !listQuery.data || listQuery.data.configs.length === 0 ? (
            <div className="flex flex-col items-center gap-3 py-8 text-center text-muted-foreground">
              <FilePlus2 className="h-8 w-8" aria-hidden />
              <p className="text-sm">{t("configs.empty")}</p>
            </div>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>{t("common.name")}</TableHead>
                  <TableHead className="w-[1%] text-right pr-3">
                    {t("common.actions")}
                  </TableHead>
                </TableRow>
              </TableHeader>
              <TableBody>
                {listQuery.data.configs.map((id) => (
                  <TableRow
                    key={id}
                    className="cursor-pointer"
                    onClick={() => navigate(`/configs/${encodeURIComponent(id)}`)}
                  >
                    <TableCell className="font-medium">{id}</TableCell>
                    <TableCell className="text-right pr-3">
                      <div
                        className="flex justify-end gap-2"
                        onClick={(event) => event.stopPropagation()}
                      >
                        <Button
                          variant="outline"
                          size="sm"
                          onClick={() =>
                            navigate(`/configs/${encodeURIComponent(id)}`)
                          }
                        >
                          {t("configs.openDetail")}
                        </Button>
                        <Button
                          variant="ghost"
                          size="icon"
                          onClick={() => onDelete(id)}
                          aria-label={t("common.delete")}
                          disabled={deleteMutation.isPending}
                        >
                          <Trash2 className="h-4 w-4" aria-hidden />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>

      <CreateConfigDialog
        open={createOpen}
        onOpenChange={setCreateOpen}
        onCreated={(id) => {
          void queryClient.invalidateQueries({ queryKey: ["configs"] });
          navigate(`/configs/${encodeURIComponent(id)}`);
        }}
      />
      <UploadFileDialog open={uploadOpen} onOpenChange={setUploadOpen} />
    </div>
  );
}

interface UploadFileDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

function UploadFileDialog({ open, onOpenChange }: UploadFileDialogProps) {
  const { t } = useTranslation();
  const [file, setFile] = useState<File | null>(null);
  const [path, setPath] = useState("");
  const [overwrite, setOverwrite] = useState(false);
  const [submitting, setSubmitting] = useState(false);

  const reset = () => {
    setFile(null);
    setPath("");
    setOverwrite(false);
  };

  const onFileChange = (event: ChangeEvent<HTMLInputElement>) => {
    const nextFile = event.target.files?.[0] ?? null;
    setFile(nextFile);
    if (nextFile && path.trim() === "") {
      setPath(nextFile.name);
    }
  };

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!file || !path.trim()) {
      return;
    }
    setSubmitting(true);
    try {
      const result = await uploadFile(path.trim(), file, overwrite);
      toast.success(t("files.uploadSuccess", { path: result.path }));
      reset();
      onOpenChange(false);
    } catch (error) {
      toast.error(t("errors.generic", { message: (error as Error).message }));
    } finally {
      setSubmitting(false);
    }
  };

  return (
    <Dialog
      open={open}
      onOpenChange={(nextOpen) => {
        onOpenChange(nextOpen);
        if (!nextOpen) {
          reset();
        }
      }}
    >
      <DialogContent>
        <form onSubmit={onSubmit} className="space-y-4">
          <DialogHeader>
            <DialogTitle>{t("files.uploadTitle")}</DialogTitle>
            <DialogDescription>{t("files.uploadDescription")}</DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="upload-file">{t("files.fileLabel")}</Label>
            <Input
              id="upload-file"
              type="file"
              accept=".yaml,.yml"
              onChange={onFileChange}
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="upload-path">{t("files.pathLabel")}</Label>
            <Input
              id="upload-path"
              value={path}
              onChange={(event) => setPath(event.target.value)}
              placeholder={t("files.pathPlaceholder")}
              required
            />
            <p className="text-xs text-muted-foreground">
              {t("files.pathHelp")}
            </p>
          </div>
          <label className="flex items-center gap-2 text-sm">
            <input
              type="checkbox"
              checked={overwrite}
              onChange={(event) => setOverwrite(event.target.checked)}
              className="h-4 w-4 shrink-0 rounded border-input"
            />
            <span>{t("files.overwrite")}</span>
          </label>
          <DialogFooter>
            <Button
              type="button"
              variant="ghost"
              onClick={() => onOpenChange(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button type="submit" disabled={submitting || !file}>
              {submitting ? t("common.loading") : t("files.upload")}
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  );
}

interface CreateConfigDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onCreated: (id: string) => void;
}

function CreateConfigDialog({
  open,
  onOpenChange,
  onCreated,
}: CreateConfigDialogProps) {
  const { t } = useTranslation();
  const [id, setId] = useState("");
  const [template, setTemplate] = useState("template.yaml");
  const [strategy, setStrategy] = useState<RulesetStrategy>("");
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    if (!id.trim() || !template.trim()) {
      return;
    }
    setSubmitting(true);
    try {
      const rule: MergeRule = {
        template: template.trim(),
        configurations: [],
        rulesetStrategy: strategy,
      };
      await createConfig(id.trim(), rule);
      toast.success(t("configs.createSuccess", { id: id.trim() }));
      onCreated(id.trim());
      onOpenChange(false);
      setId("");
      setTemplate("template.yaml");
      setStrategy("");
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
            <DialogTitle>{t("configs.createTitle")}</DialogTitle>
            <DialogDescription>
              {t("configs.createDescription")}
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2">
            <Label htmlFor="config-id">{t("configs.idLabel")}</Label>
            <Input
              id="config-id"
              value={id}
              onChange={(event) => setId(event.target.value)}
              placeholder={t("configs.idPlaceholder")}
              autoFocus
              required
            />
          </div>
          <div className="space-y-2">
            <Label htmlFor="config-template">
              {t("configs.templateLabel")}
            </Label>
            <Input
              id="config-template"
              value={template}
              onChange={(event) => setTemplate(event.target.value)}
              placeholder={t("configs.templatePlaceholder")}
              required
            />
          </div>
          <div className="space-y-2">
            <Label>{t("configs.rulesetStrategyLabel")}</Label>
            <Select
              value={strategy === "" ? "__empty" : strategy}
              onValueChange={(value) =>
                setStrategy(
                  value === "__empty" ? "" : (value as RulesetStrategy),
                )
              }
            >
              <SelectTrigger>
                <SelectValue placeholder="—" />
              </SelectTrigger>
              <SelectContent>
                {RULESET_STRATEGIES.map((option) => (
                  <SelectItem
                    key={option.label}
                    value={option.value === "" ? "__empty" : option.value}
                  >
                    {option.label}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
            <p className="text-xs text-muted-foreground">
              {t("configs.rulesetStrategyHelp")}
            </p>
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
