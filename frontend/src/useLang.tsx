import { createContext, useCallback, useContext, useEffect, useState, type ReactNode } from "react";
import { translate, type Lang, type TranslationKey } from "./i18n";

const STORAGE_KEY = "job-hunter-agent-lang";

function detectInitialLang(): Lang {
  const saved = localStorage.getItem(STORAGE_KEY);
  if (saved === "en" || saved === "zh") {
    return saved;
  }
  return navigator.language.startsWith("zh") ? "zh" : "en";
}

type LangContextValue = {
  lang: Lang;
  setLang: (lang: Lang) => void;
  toggleLang: () => void;
  t: (key: TranslationKey, params?: Record<string, string | number>) => string;
};

const LangContext = createContext<LangContextValue | null>(null);

export function LangProvider({ children }: { children: ReactNode }) {
  const [lang, setLangState] = useState<Lang>(detectInitialLang);

  useEffect(() => {
    localStorage.setItem(STORAGE_KEY, lang);
    document.documentElement.lang = lang === "zh" ? "zh-CN" : "en";
  }, [lang]);

  const setLang = useCallback((next: Lang) => setLangState(next), []);

  const toggleLang = useCallback(() => {
    setLangState((prev) => (prev === "en" ? "zh" : "en"));
  }, []);

  const t = useCallback(
    (key: TranslationKey, params?: Record<string, string | number>) => translate(lang, key, params),
    [lang],
  );

  return (
    <LangContext.Provider value={{ lang, setLang, toggleLang, t }}>
      {children}
    </LangContext.Provider>
  );
}

export function useLang() {
  const ctx = useContext(LangContext);
  if (!ctx) {
    throw new Error("useLang must be used within a LangProvider");
  }
  return ctx;
}
