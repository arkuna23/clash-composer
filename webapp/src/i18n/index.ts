import i18n from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";
import en from "./locales/en.json";
import zh from "./locales/zh.json";

export const SUPPORTED_LANGUAGES = ["zh", "en"] as const;
export type SupportedLanguage = (typeof SUPPORTED_LANGUAGES)[number];

void i18n
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: "zh",
    supportedLngs: SUPPORTED_LANGUAGES as unknown as string[],
    interpolation: { escapeValue: false },
    resources: {
      en: { translation: en },
      zh: { translation: zh },
    },
    detection: {
      order: ["localStorage", "navigator"],
      lookupLocalStorage: "clash-composer.lang",
      caches: ["localStorage"],
    },
  });

export default i18n;
