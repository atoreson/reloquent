interface ExplanationCardProps {
  category: string;
  summary: string;
  detail: string;
}

export function ExplanationCard({
  category,
  summary,
  detail,
}: ExplanationCardProps) {
  return (
    <div className="rounded-lg border border-gray-200 bg-white p-4">
      <div className="flex items-center gap-2 mb-1">
        <span className="text-xs font-medium text-blue-700 bg-blue-50 px-2 py-0.5 rounded-full">
          {category}
        </span>
      </div>
      <p className="text-sm font-medium text-gray-900">{summary}</p>
      <p className="mt-1 text-sm text-gray-600">{detail}</p>
    </div>
  );
}
