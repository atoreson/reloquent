interface CostEstimateProps {
  label: string;
  costLow: number;
  costHigh: number;
  detail?: string;
}

export function CostEstimate({
  label,
  costLow,
  costHigh,
  detail,
}: CostEstimateProps) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <p className="text-sm text-gray-600">{label}</p>
      <p className="mt-1 text-2xl font-bold text-gray-900">
        ${costLow.toFixed(2)} – ${costHigh.toFixed(2)}
      </p>
      {detail && <p className="mt-1 text-xs text-gray-500">{detail}</p>}
    </div>
  );
}
