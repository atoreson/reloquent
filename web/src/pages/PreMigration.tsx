import { Button } from "../components/Button";
import { StatusBadge } from "../components/StatusBadge";
import { PageContainer } from "../components/PageContainer";
import { useNavigateToStep } from "../api/hooks";

interface Check {
  name: string;
  status: "pass" | "pending" | "fail";
  detail: string;
}

export default function PreMigration() {
  const goToStep = useNavigateToStep();

  // Pre-migration checks are evaluated server-side; for now show the checklist structure
  const checks: Check[] = [
    {
      name: "Collections Created",
      status: "pending",
      detail: "Target collections will be created before migration",
    },
    {
      name: "Sharding Configured",
      status: "pending",
      detail: "If applicable, shard keys will be configured",
    },
    {
      name: "Balancer Disabled",
      status: "pending",
      detail: "Chunk balancer will be disabled during migration for performance",
    },
    {
      name: "Write Concern Set",
      status: "pending",
      detail: "Write concern set to w:1, j:false for maximum throughput",
    },
    {
      name: "PySpark Scripts Generated",
      status: "pending",
      detail: "Migration scripts generated and ready for upload",
    },
  ];

  return (
    <PageContainer>
    <div>
      <h2 className="text-2xl font-bold text-gray-900">Pre-Migration</h2>
      <p className="mt-2 text-gray-600">
        Prepare MongoDB and generate migration artifacts before starting.
      </p>

      <div className="mt-6 rounded-lg border border-gray-200 bg-white divide-y divide-gray-100">
        {checks.map((check) => (
          <div
            key={check.name}
            className="flex items-center justify-between px-4 py-3"
          >
            <div>
              <p className="text-sm font-medium text-gray-900">{check.name}</p>
              <p className="text-xs text-gray-500 mt-0.5">{check.detail}</p>
            </div>
            <StatusBadge status={check.status} />
          </div>
        ))}
      </div>

      <div className="mt-6 flex gap-3">
        <Button onClick={() => goToStep("migration")}>
          Continue to Migration
        </Button>
      </div>
    </div>
    </PageContainer>
  );
}
