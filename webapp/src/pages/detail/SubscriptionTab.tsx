import { useTranslation } from "react-i18next";
import { Copy, Download } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { buildSubscriptionURL } from "@/api/client";
import { copyToClipboard } from "@/lib/clipboard";
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
      await copyToClipboard(url);
      toast.success(t("common.copied"));
    } catch (error) {
      toast.error((error as Error).message);
    }
  };

  return (
    <div className="space-y-4">
      <div className="space-y-2">
        <Label htmlFor="subscription-url">URL</Label>
        <Input
          id="subscription-url"
          name="subscriptionUrl"
          type="url"
          readOnly
          value={url}
          onFocus={(event) => event.currentTarget.select()}
          className="font-mono text-xs"
          spellCheck={false}
          translate="no"
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
        {url ? (
          <Button variant="outline" className="gap-1.5" asChild>
            <a href={url} target="_blank" rel="noopener noreferrer">
              <Download className="h-4 w-4" aria-hidden />
              {t("subscription.downloadButton")}
            </a>
          </Button>
        ) : (
          <Button variant="outline" className="gap-1.5" disabled>
            <Download className="h-4 w-4" aria-hidden />
            {t("subscription.downloadButton")}
          </Button>
        )}
      </div>
    </div>
  );
}
