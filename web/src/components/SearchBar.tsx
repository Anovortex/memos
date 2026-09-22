import { SearchIcon, SparklesIcon } from "lucide-react";
import { useRef, useState } from "react";
import { useMemoFilterContext } from "@/contexts/MemoFilterContext";
import { cn } from "@/lib/utils";
import { useTranslate } from "@/utils/i18n";
import MemoDisplaySettingMenu from "./MemoDisplaySettingMenu";

const SearchBar = () => {
  const t = useTranslate();
  const { addFilter, removeFiltersByFactor } = useMemoFilterContext();
  const [queryText, setQueryText] = useState("");
  const [semanticMode, setSemanticMode] = useState(false);
  const inputRef = useRef<HTMLInputElement>(null);

  const onTextChange = (event: React.FormEvent<HTMLInputElement>) => {
    setQueryText(event.currentTarget.value);
  };

  const onKeyDown = (e: React.KeyboardEvent<HTMLInputElement>) => {
    if (e.key === "Enter") {
      e.preventDefault();
      const trimmedText = queryText.trim();
      if (trimmedText !== "") {
        if (semanticMode) {
          // One semantic query at a time; replace any previous one.
          removeFiltersByFactor("semanticSearch");
          addFilter({
            factor: "semanticSearch",
            value: trimmedText,
          });
        } else {
          const words = trimmedText.split(/\s+/);
          words.forEach((word) => {
            addFilter({
              factor: "contentSearch",
              value: word,
            });
          });
        }
        setQueryText("");
      }
    }
  };

  return (
    <div className="relative w-full h-auto flex flex-row justify-start items-center">
      <SearchIcon className="absolute left-2 w-4 h-auto opacity-40 text-sidebar-foreground" />
      <input
        className="w-full text-sidebar-foreground leading-6 bg-sidebar border border-border text-sm rounded-lg p-1 pl-8 pr-14 outline-0"
        placeholder={semanticMode ? "Ask your notes..." : t("memo.search-placeholder")}
        value={queryText}
        onChange={onTextChange}
        onKeyDown={onKeyDown}
        ref={inputRef}
      />
      <button
        type="button"
        title="Semantic search"
        aria-pressed={semanticMode}
        onClick={() => setSemanticMode((on) => !on)}
        className={cn(
          "absolute right-8 top-1/2 -translate-y-1/2 p-0.5 rounded transition-colors",
          semanticMode ? "text-primary" : "text-sidebar-foreground opacity-40 hover:opacity-80",
        )}
      >
        <SparklesIcon className="w-4 h-auto" />
      </button>
      <MemoDisplaySettingMenu className="absolute right-2 top-2 text-sidebar-foreground" />
    </div>
  );
};

export default SearchBar;
