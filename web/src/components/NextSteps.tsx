interface NextStepsProps {
  steps: string[];
}

export function NextSteps({ steps }: NextStepsProps) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <h3 className="text-sm font-medium text-gray-700 mb-3">
        Recommended Next Steps
      </h3>
      <ol className="space-y-2">
        {steps.map((step, i) => (
          <li key={i} className="flex items-start gap-2 text-sm text-gray-600">
            <span className="flex h-5 w-5 shrink-0 items-center justify-center rounded-full bg-gray-100 text-xs font-medium text-gray-700">
              {i + 1}
            </span>
            <span>{step}</span>
          </li>
        ))}
      </ol>
    </div>
  );
}
