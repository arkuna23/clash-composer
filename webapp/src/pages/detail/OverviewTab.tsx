import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { toast } from "sonner";
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
import type { MergeRule, RulesetStrategy } from "@/api/types";

const RULESET_STRATEGIES: { value: RulesetStrategy; label: string }[] = [
  { value: "", label: "—" },
  { value: "url-ruleset", label: "url-ruleset" },
  { value: "replace-ruleset", label: "replace-ruleset" },
];

interface OverviewTabProps {
  id: string;
  rule: MergeRule;
}

export function OverviewTab({ id, rule }: OverviewTabProps) {
  const { t } = useTranslation();
  const queryClient = useQueryClient();
  const [template, setTemplate] = useState(rule.template);
  const [strategy, setStrategy] = useState<RulesetStrategy>(
    rule.rulesetStrategy ?? "",
  );

  const mutation = useMutation({
    mutationFn: () =>
      updateConfig(id, {
        ...rule,
        template: template.trim(),
        rulesetStrategy: strategy,
      }),
    onSuccess: (data) => {
      queryClient.setQueryData<MergeRule>(["configs", id], data);
      toast.success(t("detail.overviewSaved"));
    },
    onError: (error: Error) => {
      toast.error(t("errors.generic", { message: error.message }));
    },
  });

  const onSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    mutation.mutate();
  };

  return (
    <form onSubmit={onSubmit} className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="overview-template">{t("configs.templateLabel")}</Label>
        <Input
          id="overview-template"
          value={template}
          onChange={(event) => setTemplate(event.target.value)}
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
          <SelectTrigger className="max-w-xs">
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
      <Button type="submit" disabled={mutation.isPending}>
        {mutation.isPending ? t("common.loading") : t("detail.saveOverview")}
      </Button>
    </form>
  );
}
