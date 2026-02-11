import { useQuery } from "@tanstack/react-query";
import { ReadinessCheck } from "../components/ReadinessCheck";
import { NextSteps } from "../components/NextSteps";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import { api } from "../api/client";

interface ReadinessData {
  production_ready: boolean;
  checks: {
    name: string;
    passed: boolean;
    message: string;
  }[];
  next_steps: string[];
}

const DEFAULT_NEXT_STEPS = [
  "Scale down the MongoDB migration tier to the production tier",
  "Re-enable the chunk balancer (if sharded)",
  "Restore write concern to your production settings (w:majority)",
  "Update application connection strings to point to MongoDB",
  "Run application-level integration tests",
  "Set up MongoDB monitoring and alerting",
  "Configure automated backups",
  "Plan the decommissioning of the source database",
];

export default function Readiness() {
  const { data } = useQuery<ReadinessData>({
    queryKey: ["readiness"],
    queryFn: () => api.get("/api/readiness"),
    retry: false,
  });

  const checks = data?.checks || [];
  const allPassed = data?.production_ready ?? false;
  const nextSteps = data?.next_steps || DEFAULT_NEXT_STEPS;

  const handleDownloadReport = async () => {
    try {
      const res = await fetch("/api/readiness");
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "reloquent-readiness-report.json";
      a.click();
      URL.revokeObjectURL(url);
    } catch {
      // ignore
    }
  };

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900">
        Production Readiness
      </h2>
      <p className="mt-2 text-gray-600">
        Final assessment of migration completeness and production readiness.
      </p>

      <div className="mt-6 space-y-6">
        {allPassed ? (
          <div className="rounded-lg border-2 border-green-400 bg-green-50 p-6 text-center">
            <div className="text-3xl mb-2">✓</div>
            <h3 className="text-lg font-bold text-green-800">
              Production Ready
            </h3>
            <p className="text-sm text-green-700 mt-1">
              All readiness checks passed. Your migration is complete.
            </p>
          </div>
        ) : (
          <Alert type="warning">
            Some readiness checks have not passed yet. Review the items below.
          </Alert>
        )}

        {checks.length > 0 && (
          <div className="rounded-lg border border-gray-200 bg-white divide-y divide-gray-100 px-4">
            {checks.map((check) => (
              <ReadinessCheck
                key={check.name}
                name={check.name}
                passed={check.passed}
                message={check.message}
              />
            ))}
          </div>
        )}

        <NextSteps steps={nextSteps} />

        <div className="flex gap-3">
          <Button variant="secondary" onClick={handleDownloadReport}>
            Download Report
          </Button>
        </div>
      </div>
    </div>
  );
}
