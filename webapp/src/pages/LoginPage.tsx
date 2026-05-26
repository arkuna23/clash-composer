import { FormEvent, useState } from "react";
import { useTranslation } from "react-i18next";
import { Navigate, useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { Globe } from "lucide-react";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { listConfigs } from "@/api/configs";
import { ApiError } from "@/api/client";
import { useAuthToken } from "@/hooks/useAuthToken";
import { setStoredToken } from "@/stores/auth";
import { SUPPORTED_LANGUAGES, type SupportedLanguage } from "@/i18n";

export function LoginPage() {
  const { t, i18n } = useTranslation();
  const navigate = useNavigate();
  const location = useLocation();
  const { isAuthenticated } = useAuthToken();
  const [token, setToken] = useState("");
  const [submitting, setSubmitting] = useState(false);

  if (isAuthenticated) {
    const from = (location.state as { from?: string } | null)?.from;
    return <Navigate to={from ?? "/configs"} replace />;
  }

  const onSubmit = async (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const trimmed = token.trim();
    if (!trimmed) {
      toast.error(t("login.invalidToken"));
      return;
    }
    setSubmitting(true);
    // Persist first so that listConfigs() picks it up via getStoredToken().
    setStoredToken(trimmed);
    try {
      await listConfigs();
      const from = (location.state as { from?: string } | null)?.from;
      navigate(from ?? "/configs", { replace: true });
    } catch (error) {
      const message =
        error instanceof ApiError ? error.message : (error as Error).message;
      toast.error(t("login.verifyFailed", { message }));
      // ApiError 401 already cleared the token; otherwise the user may retry.
    } finally {
      setSubmitting(false);
    }
  };

  const onSwitchLanguage = (lang: SupportedLanguage) => {
    void i18n.changeLanguage(lang);
  };

  return (
    <div className="min-h-screen flex flex-col">
      <div className="flex justify-end p-4">
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <Button variant="ghost" size="sm" className="gap-1.5">
              <Globe className="h-4 w-4" aria-hidden />
              <span className="text-xs text-muted-foreground uppercase">
                {i18n.resolvedLanguage}
              </span>
            </Button>
          </DropdownMenuTrigger>
          <DropdownMenuContent align="end">
            {SUPPORTED_LANGUAGES.map((lang) => (
              <DropdownMenuItem
                key={lang}
                onSelect={() => onSwitchLanguage(lang)}
              >
                {lang === "zh" ? "中文" : "English"}
              </DropdownMenuItem>
            ))}
          </DropdownMenuContent>
        </DropdownMenu>
      </div>
      <div className="flex-1 flex items-center justify-center px-4 pb-24">
        <Card className="w-full max-w-md">
          <CardHeader>
            <CardTitle>{t("login.title")}</CardTitle>
            <CardDescription>{t("login.description")}</CardDescription>
          </CardHeader>
          <CardContent>
            <form onSubmit={onSubmit} className="space-y-4">
              <div className="space-y-2">
                <Label htmlFor="token">{t("login.tokenLabel")}</Label>
                <Input
                  id="token"
                  type="password"
                  autoComplete="current-password"
                  value={token}
                  onChange={(event) => setToken(event.target.value)}
                  placeholder={t("login.tokenPlaceholder")}
                  autoFocus
                />
              </div>
              <Button type="submit" className="w-full" disabled={submitting}>
                {submitting ? t("common.loading") : t("login.submit")}
              </Button>
            </form>
          </CardContent>
        </Card>
      </div>
    </div>
  );
}
