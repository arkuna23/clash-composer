import { Monitor, Moon, Sun } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuRadioGroup,
  DropdownMenuRadioItem,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { THEMES, type Theme, useTheme } from "./ThemeProvider";

const themeIcons = {
  light: Sun,
  dark: Moon,
  system: Monitor,
} satisfies Record<Theme, typeof Sun>;

const themeTranslationKeys = {
  light: "common.themeLight",
  dark: "common.themeDark",
  system: "common.themeSystem",
} satisfies Record<Theme, string>;

export function ThemeMenu() {
  const { t } = useTranslation();
  const { theme, setTheme } = useTheme();
  const CurrentIcon = themeIcons[theme];
  const currentLabel = t(themeTranslationKeys[theme]);
  const triggerLabel = `${t("common.theme")}: ${currentLabel}`;

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button
          variant="ghost"
          size="icon"
          className="h-9 w-9 shrink-0"
          aria-label={triggerLabel}
          title={triggerLabel}
        >
          <CurrentIcon className="h-4 w-4" aria-hidden />
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" aria-label={t("common.theme")}>
        <DropdownMenuRadioGroup
          value={theme}
          onValueChange={(value) => setTheme(value as Theme)}
        >
          {THEMES.map((option) => {
            const Icon = themeIcons[option];
            return (
              <DropdownMenuRadioItem key={option} value={option}>
                <Icon className="mr-2 h-4 w-4" aria-hidden />
                {t(themeTranslationKeys[option])}
              </DropdownMenuRadioItem>
            );
          })}
        </DropdownMenuRadioGroup>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}
