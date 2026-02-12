import { useState } from "react";
import { useQuery } from "@tanstack/react-query";
import { ProgressBar } from "../components/ProgressBar";
import { FailureDialog } from "../components/FailureDialog";
import { Alert } from "../components/Alert";
import { Button } from "../components/Button";
import { useNavigateToStep } from "../api/hooks";
import { api } from "../api/client";

interface MigrationStatus {
  phase: string;
  overall: {
    docs_written: number;
    docs_total: number;
    bytes_read: number;
    percent_complete: number;
    throughput_mbps: number;
  };
  collections: {
    name: string;
    state: string;
    docs_written: number;
    docs_total: number;
    percent_complete: number;
    error: string;
  }[];
  elapsed_time: string;
  estimated_remain: string;
  errors: string[];
}

export default function Migration() {
  const goToStep = useNavigateToStep();
  const [showFailure, setShowFailure] = useState(false);

  const { data: status } = useQuery<MigrationStatus>({
    queryKey: ["migration-status"],
    queryFn: () => api.get("/api/migration/status"),
    refetchInterval: 2000,
    retry: false,
  });

  const failedCollections =
    status?.collections
      .filter((c) => c.state === "failed")
      .map((c) => c.name) || [];

  const isComplete = status?.phase === "complete";
  const hasFailed = failedCollections.length > 0;

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900">Migration</h2>
      <p className="mt-2 text-gray-600">
        {isComplete
          ? "Migration complete."
          : "Migration in progress. You can safely close this browser — the migration continues server-side."}
      </p>

      {status && (
        <div className="mt-6 space-y-6">
          <div className="rounded-lg border border-gray-200 bg-white p-4">
            <div className="flex items-center justify-between mb-3">
              <h3 className="text-sm font-medium text-gray-700">
                Overall Progress
              </h3>
              <span className="text-xs text-gray-500">
                {status.overall.throughput_mbps.toFixed(1)} MB/s
              </span>
            </div>
            <ProgressBar
              percent={status.overall.percent_complete}
              label={`${status.overall.docs_written.toLocaleString()} / ${status.overall.docs_total.toLocaleString()} documents`}
              color={isComplete ? "green" : "blue"}
              animated={!isComplete}
            />
            <div className="mt-2 flex justify-between text-xs text-gray-500">
              <span>Elapsed: {status.elapsed_time}</span>
              <span>Remaining: {status.estimated_remain || "—"}</span>
            </div>
          </div>

          <div className="space-y-2">
            <h3 className="text-sm font-medium text-gray-700">
              Per-Collection
            </h3>
            {status.collections.map((col) => (
              <div
                key={col.name}
                className="rounded-lg border border-gray-100 bg-white p-3"
              >
                <div className="flex items-center justify-between mb-1">
                  <span className="text-sm font-mono text-gray-900">
                    {col.name}
                  </span>
                  <span
                    className={`text-xs px-2 py-0.5 rounded-full ${
                      col.state === "complete"
                        ? "bg-green-100 text-green-700"
                        : col.state === "failed"
                          ? "bg-red-100 text-red-700"
                          : col.state === "running"
                            ? "bg-blue-100 text-blue-700"
                            : "bg-gray-100 text-gray-600"
                    }`}
                  >
                    {col.state}
                  </span>
                </div>
                <ProgressBar
                  percent={col.percent_complete}
                  color={
                    col.state === "complete"
                      ? "green"
                      : col.state === "failed"
                        ? "red"
                        : "blue"
                  }
                />
                {col.error && (
                  <p className="mt-1 text-xs text-red-600">{col.error}</p>
                )}
              </div>
            ))}
          </div>

          {status.errors && status.errors.length > 0 && (
            <div className="space-y-1">
              {status.errors.map((err, i) => (
                <Alert key={i} type="error">
                  {err}
                </Alert>
              ))}
            </div>
          )}

          <div className="flex gap-3">
            {hasFailed && !isComplete && (
              <Button
                variant="danger"
                onClick={() => setShowFailure(true)}
              >
                Handle Failures
              </Button>
            )}
            {isComplete && (
              <Button onClick={() => goToStep("validation")}>
                Continue to Validation
              </Button>
            )}
          </div>
        </div>
      )}

      {!status && (
        <div className="mt-6">
          <Alert type="info">
            Waiting for migration status... The migration may not have started
            yet.
          </Alert>
        </div>
      )}

      <FailureDialog
        open={showFailure}
        failedCollections={failedCollections}
        onRetry={() => {
          api.post("/api/migration/retry");
          setShowFailure(false);
        }}
        onAbort={() => {
          api.post("/api/migration/abort");
          setShowFailure(false);
        }}
        onClose={() => setShowFailure(false)}
      />
    </div>
  );
}
