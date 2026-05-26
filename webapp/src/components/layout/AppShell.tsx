import { useTranslation } from "react-i18next";
import { Globe, LogOut } from "lucide-react";
import { Link, Outlet, useLocation, useNavigate } from "react-router-dom";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { useAuthToken } from "@/hooks/useAuthToken";
import { SUPPORTED_LANGUAGES, type SupportedLanguage } from "@/i18n";

export function AppShell() {
  const { t, i18n } = useTranslation();
  const location = useLocation();
  const navigate = useNavigate();
  const { clearToken } = useAuthToken();

  const onLogout = () => {
    clearToken();
    navigate("/login", { replace: true });
  };

  const onSwitchLanguage = (lang: SupportedLanguage) => {
    void i18n.changeLanguage(lang);
  };

  const showHome = location.pathname !== "/configs";

  return (
    <div className="min-h-screen flex flex-col">
      <header className="border-b bg-background/80 backdrop-blur sticky top-0 z-30">
        <div className="container flex h-14 items-center justify-between gap-4">
          <Link to="/configs" className="flex items-center gap-2">
            <span className="inline-flex h-7 w-7 items-center justify-center rounded-md bg-primary text-primary-foreground text-xs font-semibold">
              CC
            </span>
            <div className="flex flex-col leading-tight">
              <span className="text-sm font-semibold">{t("app.title")}</span>
              <span className="text-xs text-muted-foreground hidden sm:block">
                {t("app.subtitle")}
              </span>
            </div>
          </Link>
          <div className="flex items-center gap-2">
            {showHome && (
              <Button variant="ghost" size="sm" asChild>
                <Link to="/configs">{t("configs.title")}</Link>
              </Button>
            )}
            <DropdownMenu>
              <DropdownMenuTrigger asChild>
                <Button variant="ghost" size="sm" className="gap-1.5">
                  <Globe className="h-4 w-4" aria-hidden />
                  <span className="hidden sm:inline">
                    {t("common.language")}
                  </span>
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
            <Button
              variant="ghost"
              size="sm"
              onClick={onLogout}
              className="gap-1.5"
            >
              <LogOut className="h-4 w-4" aria-hidden />
              <span className="hidden sm:inline">{t("common.logout")}</span>
            </Button>
          </div>
        </div>
      </header>
      <main className="container flex-1 py-6">
        <Outlet />
      </main>
      <footer className="border-t py-3 text-center text-xs text-muted-foreground">
        <span>Clash Composer · webapp</span>
      </footer>
    </div>
  );
}
