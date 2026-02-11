import { Button } from "./Button";

interface FailureDialogProps {
  open: boolean;
  failedCollections: string[];
  onRetry: () => void;
  onAbort: () => void;
  onClose: () => void;
  retryLoading?: boolean;
  abortLoading?: boolean;
}

export function FailureDialog({
  open,
  failedCollections,
  onRetry,
  onAbort,
  onClose,
  retryLoading,
  abortLoading,
}: FailureDialogProps) {
  if (!open) return null;

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center">
      <div className="absolute inset-0 bg-black/50" onClick={onClose} />
      <div className="relative bg-white rounded-lg shadow-xl max-w-md w-full mx-4 p-6">
        <h3 className="text-lg font-semibold text-red-600">
          Partial Migration Failure
        </h3>
        <p className="mt-2 text-sm text-gray-600">
          The following collections failed to migrate:
        </p>
        <ul className="mt-2 space-y-1">
          {failedCollections.map((name) => (
            <li
              key={name}
              className="text-sm font-mono text-red-700 bg-red-50 px-2 py-1 rounded"
            >
              {name}
            </li>
          ))}
        </ul>
        <div className="mt-6 flex justify-end gap-3">
          <Button
            variant="danger"
            onClick={onAbort}
            loading={abortLoading}
            disabled={retryLoading}
          >
            Abort Migration
          </Button>
          <Button
            variant="primary"
            onClick={onRetry}
            loading={retryLoading}
            disabled={abortLoading}
          >
            Retry Failed
          </Button>
        </div>
      </div>
    </div>
  );
}
