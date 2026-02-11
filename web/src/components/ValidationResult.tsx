import { StatusBadge } from "./StatusBadge";

interface ValidationResultCardProps {
  collection: string;
  rowCountPassed?: boolean;
  samplePassed?: boolean;
  aggregatePassed?: boolean;
  status: string;
}

export function ValidationResultCard({
  collection,
  rowCountPassed,
  samplePassed,
  aggregatePassed,
  status,
}: ValidationResultCardProps) {
  const overallStatus =
    status === "pass" ? "pass" : status === "fail" ? "fail" : "pending";

  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <div className="flex items-center justify-between mb-3">
        <h4 className="text-sm font-medium text-gray-900 font-mono">
          {collection}
        </h4>
        <StatusBadge
          status={overallStatus as "pass" | "fail" | "pending"}
          label={status}
        />
      </div>
      <div className="grid grid-cols-3 gap-2">
        <Check label="Row Count" passed={rowCountPassed} />
        <Check label="Sample" passed={samplePassed} />
        <Check label="Aggregate" passed={aggregatePassed} />
      </div>
    </div>
  );
}

function Check({
  label,
  passed,
}: {
  label: string;
  passed?: boolean;
}) {
  return (
    <div className="text-center">
      <div
        className={`text-lg ${
          passed === undefined
            ? "text-gray-300"
            : passed
              ? "text-green-500"
              : "text-red-500"
        }`}
      >
        {passed === undefined ? "○" : passed ? "✓" : "✗"}
      </div>
      <div className="text-xs text-gray-500">{label}</div>
    </div>
  );
}
