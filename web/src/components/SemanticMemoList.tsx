import { Code, ConnectError } from "@connectrpc/connect";
import { useQuery } from "@tanstack/react-query";
import { LoaderIcon, SearchXIcon } from "lucide-react";
import { useEffect } from "react";
import { aiServiceClient } from "@/connect";
import { useMemoFilterContext } from "@/contexts/MemoFilterContext";
import MemoView from "./MemoView";

interface SemanticMemoListProps {
  query: string;
}

// Renders semantic search results for the active semanticSearch filter. When
// the instance has no embedding provider configured, silently falls back to
// the regular substring search by swapping the filter.
const SemanticMemoList = ({ query }: SemanticMemoListProps) => {
  const { addFilter, removeFiltersByFactor } = useMemoFilterContext();

  const { data, error, isLoading } = useQuery({
    queryKey: ["semantic-search", query],
    queryFn: () => aiServiceClient.searchMemos({ query, limit: 20 }),
    retry: false,
  });

  const isUnconfigured = error instanceof ConnectError && error.code === Code.FailedPrecondition;
  useEffect(() => {
    if (!isUnconfigured) {
      return;
    }
    removeFiltersByFactor("semanticSearch");
    query
      .trim()
      .split(/\s+/)
      .forEach((word) => addFilter({ factor: "contentSearch", value: word }));
  }, [isUnconfigured, query, addFilter, removeFiltersByFactor]);

  if (isLoading || isUnconfigured) {
    return (
      <div className="w-full flex flex-row justify-center items-center py-8 text-muted-foreground">
        <LoaderIcon className="w-5 h-auto animate-spin" />
      </div>
    );
  }
  if (error) {
    return <div className="w-full py-8 text-center text-sm text-destructive">{error.message}</div>;
  }
  if (!data || data.results.length === 0) {
    return (
      <div className="w-full flex flex-col justify-center items-center gap-2 py-10 text-muted-foreground">
        <SearchXIcon className="w-6 h-auto" />
        <span className="text-sm">No matching memos</span>
      </div>
    );
  }

  return (
    <div className="w-full flex flex-col justify-start items-start gap-3">
      {data.results.map(
        (result) => result.memo && <MemoView key={result.memo.name} memo={result.memo} showVisibility showPinned compact={false} />,
      )}
    </div>
  );
};

export default SemanticMemoList;
