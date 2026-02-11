import { useState } from "react";
import { Button } from "../components/Button";
import { Alert } from "../components/Alert";
import { ConfirmDialog } from "../components/ConfirmDialog";
import { useWizardState, useSetStep } from "../api/hooks";

export default function Review() {
  const { data: state } = useWizardState();
  const setStep = useSetStep();
  const [showConfirm, setShowConfirm] = useState(false);

  const handleStartMigration = () => {
    setStep.mutate("migration");
    setShowConfirm(false);
  };

  const completedSteps = state
    ? Object.entries(state.steps).filter(([, s]) => s.status === "complete")
        .length
    : 0;

  return (
    <div>
      <h2 className="text-2xl font-bold text-gray-900">Review</h2>
      <p className="mt-2 text-gray-600">
        Review the complete migration plan before starting.
      </p>

      <div className="mt-6 space-y-4">
        <div className="rounded-lg border border-gray-200 bg-white p-4">
          <h3 className="text-sm font-medium text-gray-700 mb-3">
            Migration Summary
          </h3>
          <dl className="grid grid-cols-2 gap-x-4 gap-y-2 text-sm">
            <dt className="text-gray-500">Steps Completed</dt>
            <dd className="font-medium">{completedSteps} / 12</dd>
            <dt className="text-gray-500">Current Step</dt>
            <dd className="font-medium">{state?.current_step || "—"}</dd>
          </dl>
        </div>

        <Alert type="warning">
          Starting the migration is a point of no return. You will not be able
          to navigate back to configuration steps once the migration begins.
          Ensure all settings are correct before proceeding.
        </Alert>

        <div className="flex gap-3">
          <Button
            variant="primary"
            onClick={() => setShowConfirm(true)}
          >
            Start Migration
          </Button>
        </div>
      </div>

      <ConfirmDialog
        open={showConfirm}
        title="Start Migration"
        message="Are you sure you want to start the migration? This action cannot be undone. The migration will begin executing PySpark jobs against your source database and writing to MongoDB."
        confirmLabel="Start Migration"
        variant="danger"
        onConfirm={handleStartMigration}
        onCancel={() => setShowConfirm(false)}
      />
    </div>
  );
}
