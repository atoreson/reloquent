interface ReadinessCheckProps {
  name: string;
  passed: boolean;
  message: string;
}

export function ReadinessCheck({ name, passed, message }: ReadinessCheckProps) {
  return (
    <div className="flex items-start gap-3 py-3">
      <div
        className={`mt-0.5 flex h-5 w-5 shrink-0 items-center justify-center rounded-full text-xs font-bold ${
          passed
            ? "bg-green-100 text-green-600"
            : "bg-red-100 text-red-600"
        }`}
      >
        {passed ? "✓" : "✗"}
      </div>
      <div>
        <p className="text-sm font-medium text-gray-900">{name}</p>
        <p className="text-xs text-gray-500 mt-0.5">{message}</p>
      </div>
    </div>
  );
}
