import { describe, expect, it } from "vitest";
import { buildQuickFindFilters, readQuickFindQuery } from "@/components/AppSidebar/QuickFindDialog";
import type { MemoFilter } from "@/contexts/MemoFilterContext";

describe("semantic quick find", () => {
  it("submits one semanticSearch filter and replaces the previous one, keeping the scope", () => {
    const previous = [
      { factor: "semanticSearch", value: "old question" },
      { factor: "tagSearch", value: "work" },
    ] as MemoFilter[];

    expect(buildQuickFindFilters("what did I plan", previous, true, "semantic")).toEqual([
      { factor: "tagSearch", value: "work" },
      { factor: "semanticSearch", value: "what did I plan" },
    ]);
  });

  it("submits nothing for a blank question", () => {
    expect(buildQuickFindFilters("   ", [], true, "semantic")).toEqual([]);
  });

  it("reads a semantic filter back as semantic mode", () => {
    expect(readQuickFindQuery([{ factor: "semanticSearch", value: "plans" }] as MemoFilter[])).toEqual({
      query: "plans",
      mode: "semantic",
    });
  });
});
