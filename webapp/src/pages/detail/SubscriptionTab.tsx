import { useTranslation } from "react-i18next";
import { Copy, Download } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { buildSubscriptionURL } from "@/api/client";
import { useAuthToken } from "@/hooks/useAuthToken";

interface SubscriptionTabProps {
  id: string;
}

export function SubscriptionTab({ id }: SubscriptionTabProps) {
  const { t } = useTranslation();
  const { token } = useAuthToken();
  const url = token ? buildSubscriptionURL(id, token) : "";

  const onCopy = async () => {
    if (!url) return;
    try {
      await navigator.clipboard.writeText(url);
      toast.success(t("common.copied"));
    } catch (error) {
      toast.error((error as Error).message);
    }
  };

  const onDownload = () => {
    if (!url) return;
    window.open(url, "_blank", "noopener,noreferrer");
  };

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="subscription-url">URL</Label>
        <Input
          id="subscription-url"
          readOnly
          value={url}
          onFocus={(event) => event.currentTarget.select()}
          className="font-mono text-xs"
        />
        <p className="text-xs text-muted-foreground">
          {t("subscription.copyHint")}
        </p>
      </div>
      <div className="flex flex-wrap gap-2">
        <Button onClick={onCopy} className="gap-1.5" disabled={!url}>
          <Copy className="h-4 w-4" aria-hidden />
          {t("subscription.copyButton")}
        </Button>
        <Button
          onClick={onDownload}
          variant="outline"
          className="gap-1.5"
          disabled={!url}
        >
          <Download className="h-4 w-4" aria-hidden />
          {t("subscription.downloadButton")}
        </Button>
      </div>
    </div>
  );
}
