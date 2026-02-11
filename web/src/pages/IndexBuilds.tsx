import { useQuery } from "@tanstack/react-query";
import { ProgressBar } from "../components/ProgressBar";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import { useSetStep } from "../api/hooks";
import { api } from "../api/client";

interface IndexStatus {
  collection: string;
  index_name: string;
  phase: string;
  progress: number;
  message: string;
}

export default function IndexBuilds() {
  const setStep = useSetStep();

  const { data: indexes } = useQuery<IndexStatus[]>({
    queryKey: ["index-status"],
    queryFn: () => api.get("/api/indexes/status"),
    refetchInterval: 2000,
    retry: false,
  });

  const allComplete =
    indexes && indexes.length > 0 && indexes.every((i) => i.phase === "complete");

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900">Index Builds</h2>
      <p className="mt-2 text-gray-600">
        Building indexes on the migrated collections.
      </p>

      {indexes && indexes.length > 0 && (
        <div className="mt-6 space-y-3">
          {indexes.map((idx) => (
            <div
              key={`${idx.collection}-${idx.index_name}`}
              className="rounded-lg border border-gray-200 bg-white p-4"
            >
              <div className="flex items-center justify-between mb-2">
                <div>
                  <span className="text-sm font-mono text-gray-900">
                    {idx.collection}
                  </span>
                  <span className="text-xs text-gray-500 ml-2">
                    {idx.index_name}
                  </span>
                </div>
                <span
                  className={`text-xs px-2 py-0.5 rounded-full ${
                    idx.phase === "complete"
                      ? "bg-green-100 text-green-700"
                      : idx.phase === "building"
                        ? "bg-blue-100 text-blue-700"
                        : "bg-gray-100 text-gray-600"
                  }`}
                >
                  {idx.phase}
                </span>
              </div>
              <ProgressBar
                percent={idx.progress}
                color={idx.phase === "complete" ? "green" : "blue"}
                animated={idx.phase === "building"}
              />
              {idx.message && (
                <p className="mt-1 text-xs text-gray-500">{idx.message}</p>
              )}
            </div>
          ))}
        </div>
      )}

      {indexes && indexes.length === 0 && (
        <Alert type="info">No indexes to build.</Alert>
      )}

      {!indexes && (
        <div className="mt-6">
          <Alert type="info">Loading index build status...</Alert>
        </div>
      )}

      {allComplete && (
        <div className="mt-6 flex gap-3">
          <Button onClick={() => setStep.mutate("complete")}>
            Continue to Readiness
          </Button>
        </div>
      )}
    </div>
  );
}
