import { FormEvent, useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
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
import { updateConfig } from "@/api/configs";
import type { MergeRule, RulesetStrategy } from "@/api/types";

const RULESET_STRATEGIES: { value: RulesetStrategy; label: string }[] = [
  { value: "", label: "—" },
  { value: "url-ruleset", label: "url-ruleset" },
  { value: "replace-ruleset", label: "replace-ruleset" },
];

interface OverviewTabProps {
  id: string;
  rule: MergeRule;
  onDirtyChange?: (dirty: boolean) => void;
}

export function OverviewTab({ id, rule, onDirtyChange }: OverviewTabProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [template, setTemplate] = useState(rule.template);
  const [strategy, setStrategy] = useState<RulesetStrategy>(
    rule.rulesetStrategy ?? "",
  );
  const [cacheDurationSeconds, setCacheDurationSeconds] = useState(
    String(rule.cacheDurationSeconds ?? 0),
  );
  const [error, setError] = useState("");

  const isDirty =
    template !== rule.template ||
    strategy !== (rule.rulesetStrategy ?? "") ||
    cacheDurationSeconds !== String(rule.cacheDurationSeconds ?? 0);

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
    setTemplate(rule.template);
    setStrategy(rule.rulesetStrategy ?? "");
    setCacheDurationSeconds(String(rule.cacheDurationSeconds ?? 0));
  }, [rule]);

  const mutation = useMutation({
    mutationFn: () =>
      updateConfig(id, {
        ...rule,
        template: template.trim(),
        rulesetStrategy: strategy,
        cacheDurationSeconds: Math.max(
          0,
          Math.floor(Number(cacheDurationSeconds) || 0),
        ),
      }),
    onSuccess: (data) => {
      queryClient.setQueryData<MergeRule>(["configs", id], data);
      setError("");
      toast.success(t("detail.overviewSaved"));
    },
    onError: (error: Error) => {
      setError(t("errors.generic", { message: error.message }));
    },
  });

  const onSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError("");
    mutation.mutate();
  };

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="overview-template">{t("configs.templateLabel")}</Label>
        <Input
          id="overview-template"
          name="template"
          value={template}
          onChange={(event) => {
            setTemplate(event.target.value);
            setError("");
          }}
          autoComplete="off"
          spellCheck={false}
          required
        />
      </div>
      <div className="space-y-2">
        <Label htmlFor="overview-ruleset-strategy">
          {t("configs.rulesetStrategyLabel")}
        </Label>
        <Select
          value={strategy === "" ? "__empty" : strategy}
          onValueChange={(value) =>
            setStrategy(
              value === "__empty" ? "" : (value as RulesetStrategy),
            )
          }
        >
          <SelectTrigger
            id="overview-ruleset-strategy"
            name="rulesetStrategy"
            className="max-w-xs"
          >
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
      </div>
      <div className="space-y-2">
        <Label htmlFor="overview-cache-duration">
          {t("configs.cacheDurationLabel")}
        </Label>
        <Input
          id="overview-cache-duration"
          name="cacheDurationSeconds"
          type="number"
          inputMode="numeric"
          min={0}
          step={1}
          value={cacheDurationSeconds}
          onChange={(event) => setCacheDurationSeconds(event.target.value)}
          autoComplete="off"
          className="max-w-xs"
        />
        <p className="text-xs text-muted-foreground">
          {t("configs.cacheDurationHelp")}
        </p>
      </div>
      <FormError id="overview-error" message={error} />
      <Button
        type="submit"
        disabled={mutation.isPending}
        aria-busy={mutation.isPending}
      >
        {mutation.isPending ? t("common.loading") : t("detail.saveOverview")}
      </Button>
    </form>
  );
}
