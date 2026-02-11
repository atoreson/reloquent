import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import { ExplanationCard } from "../components/ExplanationCard";
import { CostEstimate } from "../components/CostEstimate";
import { useSizing, useSetStep } from "../api/hooks";

export default function Sizing() {
  const { data: plan, isLoading, error } = useSizing();
  const setStep = useSetStep();

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-64">
        <div className="animate-spin h-8 w-8 border-4 border-blue-500 border-t-transparent rounded-full" />
      </div>
    );
  }

  if (error) {
    return <Alert type="error">{error.message}</Alert>;
  }

  if (!plan) return null;

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900">Sizing</h2>
      <p className="mt-2 text-gray-600">
        Review cluster sizing recommendations and cost estimates.
      </p>

      <div className="mt-6 grid grid-cols-2 gap-4">
        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-3">
            Spark Cluster
          </h3>
          <dl className="space-y-2 text-sm">
            <div className="flex justify-between">
              <dt className="text-gray-500">Platform</dt>
              <dd className="font-medium">{plan.spark_plan.platform.toUpperCase()}</dd>
            </div>
            {plan.spark_plan.instance_type && (
              <div className="flex justify-between">
                <dt className="text-gray-500">Instance Type</dt>
                <dd className="font-medium">{plan.spark_plan.instance_type}</dd>
              </div>
            )}
            {plan.spark_plan.worker_count > 0 && (
              <div className="flex justify-between">
                <dt className="text-gray-500">Workers</dt>
                <dd className="font-medium">{plan.spark_plan.worker_count}</dd>
              </div>
            )}
            {plan.spark_plan.dpu_count > 0 && (
              <div className="flex justify-between">
                <dt className="text-gray-500">DPUs (Glue)</dt>
                <dd className="font-medium">{plan.spark_plan.dpu_count}</dd>
              </div>
            )}
          </dl>
        </div>

        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-3">
            MongoDB
          </h3>
          <dl className="space-y-2 text-sm">
            <div className="flex justify-between">
              <dt className="text-gray-500">Migration Tier</dt>
              <dd className="font-medium">{plan.mongo_plan.migration_tier}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Production Tier</dt>
              <dd className="font-medium">{plan.mongo_plan.production_tier}</dd>
            </div>
            <div className="flex justify-between">
              <dt className="text-gray-500">Storage</dt>
              <dd className="font-medium">{plan.mongo_plan.storage_gb} GB</dd>
            </div>
          </dl>
        </div>
      </div>

      <div className="mt-4 grid grid-cols-2 gap-4">
        <CostEstimate
          label="Estimated Migration Cost"
          costLow={plan.spark_plan.cost_low}
          costHigh={plan.spark_plan.cost_high}
          detail={plan.spark_plan.cost_estimate}
        />
        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <p className="text-sm text-gray-600">Estimated Time</p>
          <p className="mt-1 text-2xl font-bold text-gray-900">
            {plan.estimated_time}
          </p>
        </div>
      </div>

      {plan.explanations && plan.explanations.length > 0 && (
        <div className="mt-6 space-y-3">
          <h3 className="text-sm font-medium text-gray-700">Explanations</h3>
          {plan.explanations.map((exp, i) => (
            <ExplanationCard
              key={i}
              category={exp.category}
              summary={exp.summary}
              detail={exp.detail}
            />
          ))}
        </div>
      )}

      <div className="mt-6 flex gap-3">
        <Button onClick={() => setStep.mutate("aws_setup")}>
          Continue to AWS Setup
        </Button>
      </div>
    </div>
  );
}
